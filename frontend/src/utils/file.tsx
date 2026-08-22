export const getFileName = (path: string) => {
  return path.split('/').pop() || path;
};

export const isImagePath = (path: string | null | undefined) => {
  if (!path) return false;
  return /\.(jpg|jpeg|png|gif|webp)$/i.test(path);
};

const FILE_SIZE_UNITS = ['B', 'KB', 'MB', 'GB'];

// Hiển thị dung lượng file gọn cho UI: 984 B, 1.2 MB...
export const formatFileSize = (bytes: number) => {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';

  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < FILE_SIZE_UNITS.length - 1) {
    size /= 1024;
    unit += 1;
  }

  const digits = unit === 0 || size >= 100 ? 0 : 1;
  return `${size.toFixed(digits)} ${FILE_SIZE_UNITS[unit]}`;
};
