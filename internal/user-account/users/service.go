package users

import (
	"context"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/auth"
)

// UserService сервис обрабатывает запросы с пользователями
type UserService interface {
	// RegisterUser регистрация нового пользователя
	RegisterUser(ctx context.Context, newUser NewUserRequest) error
	// AuthUser аутентификация пользователя и выдача токена
	AuthUser(ctx context.Context, authRequest UserAuthRequest, secret string) (UserAuthResponse, error)
}

type UserServiceImpl struct {
	userRepo UserRepository
}

func (u UserServiceImpl) RegisterUser(ctx context.Context, newUser NewUserRequest) error {
	err := u.userRepo.SaveUser(ctx, newUser.toUser())
	if err != nil {
		return err
	}

	return nil
}

func (u UserServiceImpl) AuthUser(ctx context.Context, authRequest UserAuthRequest, secret string) (UserAuthResponse, error) {
	user, err := u.userRepo.FindByNameAndPassword(ctx, authRequest.Username, authRequest.Password)
	if err != nil {
		return UserAuthResponse{}, err
	}

	token, err := auth.GenerateToken(user.Username, secret)
	if err != nil {
		return UserAuthResponse{}, err
	}

	return UserAuthResponse{
		Token: token,
	}, nil
}

func NewUserService(userRepo UserRepository) UserServiceImpl {
	return UserServiceImpl{
		userRepo: userRepo,
	}
}
