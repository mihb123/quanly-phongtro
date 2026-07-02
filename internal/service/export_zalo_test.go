package service

import (
	"context"
	"net/http"
	"net/netip"
)

// Export unexported methods and types for testing

type ZaloServiceTesting interface {
	ZaloService
	GetDecryptedToken(ctx context.Context, managerID string) (string, error)
	ProcessTransactionImage(ctx context.Context, managerID, groupChatID, imgURL string) error
	AutoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) (bool, error)
}

func (s *zaloServiceImpl) GetDecryptedToken(ctx context.Context, managerID string) (string, error) {
	return s.getDecryptedToken(ctx, managerID)
}

func (s *zaloServiceImpl) ProcessTransactionImage(ctx context.Context, managerID, groupChatID, imgURL string) error {
	return s.processTransactionImage(ctx, managerID, groupChatID, imgURL)
}

func (s *zaloServiceImpl) AutoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) (bool, error) {
	return s.autoLinkRoom(ctx, managerID, groupChatID, groupName)
}

func CastToTesting(s ZaloService) ZaloServiceTesting {
	return s.(*zaloServiceImpl)
}

func IsZaloAuthError(err error) bool {
	return isZaloAuthError(err)
}

func GetEncryptionKey(s ZaloService) []byte {
	return s.(*zaloServiceImpl).encryptionKey
}

// SetZaloImageHTTPClientFactoryForTest replaces the image HTTP client factory during tests.
func SetZaloImageHTTPClientFactoryForTest(factory func() *http.Client) func() {
	previous := newZaloImageHTTPClient
	newZaloImageHTTPClient = factory
	return func() {
		newZaloImageHTTPClient = previous
	}
}

// SetZaloImageHostResolverForTest replaces the image host resolver during tests.
func SetZaloImageHostResolverForTest(resolver func(context.Context, string) ([]netip.Addr, error)) func() {
	previous := resolveZaloImageHost
	resolveZaloImageHost = resolver
	return func() {
		resolveZaloImageHost = previous
	}
}
