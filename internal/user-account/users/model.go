package users

// User описывает сущность пользователя
type User struct {
	Username string `db:"username"`
	Password string `db:"password"`
	TGUser   string `db:"tg_user"`
}

// NewUserRequest запрос создания нового пользователя
type NewUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	TGUser   string `json:"tg_user" validate:"required"`
}

func (r NewUserRequest) toUser() User {
	return User(r)
}

// UserAuthRequest запрос авторизации пользователя
type UserAuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UserAuthResponse ответ авторизованному пользователю
type UserAuthResponse struct {
	Token string `json:"token"`
}
