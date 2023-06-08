package users

import (
	"context"

	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UserService сервис обрабатывает запросы с пользователями
type UserService interface {
	// SetNotification устанавливает флаг отправки уведомлений пользователю
	SetNotification(ctx context.Context, tgName string, flag bool) error
	// SetUserChatID сохраняет/обновляет идентификатор чата telegram с которого обращался пользователь
	SetUserChatID(ctx context.Context, tgName string, chatID int64) error
	// FindUserByTGName поиск пользователя по имени в telegram
	FindUserByTGName(ctx context.Context, tgName string) (User, error)
	// IsUserExists проверяет, существует ли пользователь с указанным именем telegram
	IsUserExists(ctx context.Context, tgName string) bool
	// FindUserByClientID поиск пользователя по идентификатору клиентского приложения
	FindUserByClientID(ctx context.Context, clientID string) (User, error)
	SetUserLastMessage(tgName string, message tgbotapi.Message)
	GetUserLastMessage(tgName string) (tgbotapi.Message, bool)
}

type UserServiceImpl struct {
	userRepo    UserRepository
	userLastMsg *collections.ConcurrentMap[string, tgbotapi.Message]
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

func (u UserServiceImpl) FindUserByTGName(ctx context.Context, tgName string) (User, error) {
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

func (u UserServiceImpl) SetUserLastMessage(tgName string, message tgbotapi.Message) {
	u.userLastMsg.Put(tgName, message)
}

func (u UserServiceImpl) GetUserLastMessage(tgName string) (tgbotapi.Message, bool) {
	return u.userLastMsg.Get(tgName)
}

// NewUserService создает сервис пользователей
func NewUserService(userRepo UserRepository) UserServiceImpl {
	return UserServiceImpl{
		userRepo:    userRepo,
		userLastMsg: collections.NewConcurrentMap[string, tgbotapi.Message](),
	}
}
