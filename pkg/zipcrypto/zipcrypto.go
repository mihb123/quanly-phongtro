package zipcrypto

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path"
	"strings"
)

var (
	ErrPasswordRequired = errors.New("zipcrypto: archive is encrypted and needs a password")
	ErrWrongPassword    = errors.New("zipcrypto: wrong password")
	ErrUnsupported      = errors.New("zipcrypto: unsupported compression or encryption method")
	ErrTooLarge         = errors.New("zipcrypto: archive content exceeds the size limit")
)

type Limits struct {
	MaxFiles     int
	MaxFileSize  int64
	MaxTotalSize int64
}

var DefaultLimits = Limits{MaxFiles: 200, MaxFileSize: 50 << 20, MaxTotalSize: 200 << 20}

type File struct {
	Name string
	Data []byte
}

func Extract(data []byte, password string, limits Limits) ([]File, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zipcrypto: open archive: %w", err)
	}

	var files []File
	var total int64
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if limits.MaxFiles > 0 && len(files) >= limits.MaxFiles {
			return nil, ErrTooLarge
		}
		content, err := readEntry(entry, password, limits.MaxFileSize)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name, err)
		}
		total += int64(len(content))
		if limits.MaxTotalSize > 0 && total > limits.MaxTotalSize {
			return nil, ErrTooLarge
		}
		files = append(files, File{Name: path.Clean(strings.ReplaceAll(entry.Name, "\\", "/")), Data: content})
	}
	return files, nil
}

func readEntry(entry *zip.File, password string, maxSize int64) ([]byte, error) {
	if entry.Flags&0x1 == 0 {
		rc, err := entry.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return readLimited(rc, maxSize)
	}

	if password == "" {
		return nil, ErrPasswordRequired
	}
	if entry.Method == 99 || entry.Flags&0x40 != 0 {
		return nil, ErrUnsupported
	}
	raw, err := entry.OpenRaw()
	if err != nil {
		return nil, err
	}
	encrypted, err := io.ReadAll(raw)
	if err != nil {
		return nil, err
	}
	if len(encrypted) < 12 {
		return nil, ErrWrongPassword
	}

	keys := newKeys(password)
	header := keys.decrypt(append([]byte(nil), encrypted[:12]...))
	check := byte(entry.CRC32 >> 24)
	if entry.Flags&0x8 != 0 {
		check = byte(entry.ModifiedTime >> 8)
	}
	if header[11] != check {
		return nil, ErrWrongPassword
	}
	payload := keys.decrypt(append([]byte(nil), encrypted[12:]...))

	var content []byte
	switch entry.Method {
	case zip.Store:
		content = payload
		if maxSize > 0 && int64(len(content)) > maxSize {
			return nil, ErrTooLarge
		}
	case zip.Deflate:
		inflater := flate.NewReader(bytes.NewReader(payload))
		defer inflater.Close()
		if content, err = readLimited(inflater, maxSize); err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnsupported
	}
	if crc32.ChecksumIEEE(content) != entry.CRC32 {
		return nil, ErrWrongPassword
	}
	return content, nil
}

func readLimited(r io.Reader, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		return io.ReadAll(r)
	}
	content, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxSize {
		return nil, ErrTooLarge
	}
	return content, nil
}

type keys [3]uint32

func newKeys(password string) *keys {
	k := &keys{0x12345678, 0x23456789, 0x34567890}
	for i := 0; i < len(password); i++ {
		k.update(password[i])
	}
	return k
}

func (k *keys) update(b byte) {
	k[0] = crcUpdate(k[0], b)
	k[1] = (k[1]+(k[0]&0xff))*134775813 + 1
	k[2] = crcUpdate(k[2], byte(k[1]>>24))
}

func (k *keys) decrypt(buf []byte) []byte {
	for i, c := range buf {
		temp := uint16(k[2] | 2)
		plain := c ^ byte((temp*(temp^1))>>8)
		k.update(plain)
		buf[i] = plain
	}
	return buf
}

func crcUpdate(crc uint32, b byte) uint32 {
	return crc32.IEEETable[byte(crc)^b] ^ (crc >> 8)
}
