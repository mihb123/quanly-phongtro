package service

import "context"

// Export unexported methods and types for testing

type ZaloServiceTesting interface {
	ZaloService
	GetDecryptedToken(ctx context.Context, managerID string) (string, error)
	ProcessTransactionImage(ctx context.Context, managerID, groupChatID, imgURL string) error
	AutoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) error
}

func (s *zaloServiceImpl) GetDecryptedToken(ctx context.Context, managerID string) (string, error) {
	return s.getDecryptedToken(ctx, managerID)
}

func (s *zaloServiceImpl) ProcessTransactionImage(ctx context.Context, managerID, groupChatID, imgURL string) error {
	return s.processTransactionImage(ctx, managerID, groupChatID, imgURL)
}

func (s *zaloServiceImpl) AutoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) error {
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
