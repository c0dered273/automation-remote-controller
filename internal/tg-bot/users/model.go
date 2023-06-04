package users

type User struct {
	Username      string `db:"username"`
	TGUser        string `db:"tg_user"`
	ChatID        int64  `db:"chat_id"`
	NotifyEnabled bool   `db:"notify_enabled"`
}
