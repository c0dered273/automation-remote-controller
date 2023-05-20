package users

type User struct {
	Username string `db:"username"`
	Password string `db:"password"`
	TGUser   string `db:"tg_user"`
}

type NewUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	TGUser   string `json:"tg_user" validate:"required"`
}

func (r NewUserRequest) toUser() User {
	return User{
		Username: r.Username,
		Password: r.Password,
		TGUser:   r.TGUser,
	}
}

type UserAuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserAuthResponse struct {
	Token string `json:"token"`
}
