package djinn

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt Time   `json:"created_at"`
}
