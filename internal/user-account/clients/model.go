package clients

// Client описывает клиентское приложение
type Client struct {
	Name       string `db:"name"`
	ClientUUID string `db:"uuid"`
	OwnerName  string `db:"username"`
}
