package users

import "context"

type UserService interface {
	SetNotification(ctx context.Context, tgName string, flag bool) error
	SetUserChatID(ctx context.Context, tgName string, chatID int64) error
	FindUser(ctx context.Context, tgName string) (User, error)
	IsUserExists(ctx context.Context, tgName string) bool
	FindUserByClientID(ctx context.Context, clientID string) (User, error)
}

type UserServiceImpl struct {
	userRepo UserRepository
}

func (u UserServiceImpl) SetNotification(ctx context.Context, tgName string, flag bool) error {
	err := u.userRepo.UpdateNotificationByTGUser(ctx, tgName, flag)
	if err != nil {
		return err
	}
	return nil
}

func (u UserServiceImpl) SetUserChatID(ctx context.Context, tgName string, chatID int64) error {
	err := u.userRepo.UpdateChatIDByTGUser(ctx, tgName, chatID)
	if err != nil {
		return err
	}
	return nil
}

func (u UserServiceImpl) FindUser(ctx context.Context, tgName string) (User, error) {
	user, err := u.userRepo.FindUserByTGUser(ctx, tgName)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (u UserServiceImpl) IsUserExists(ctx context.Context, tgName string) bool {
	_, err := u.userRepo.IsUserExists(ctx, tgName)
	return err == nil
}

func (u UserServiceImpl) FindUserByClientID(ctx context.Context, clientID string) (User, error) {
	user, err := u.userRepo.FindUserByClientID(ctx, clientID)
	if err != nil {
		return User{}, err
	}
	return user, err
}

func NewUserService(userRepo UserRepository) UserServiceImpl {
	return UserServiceImpl{
		userRepo: userRepo,
	}
}
