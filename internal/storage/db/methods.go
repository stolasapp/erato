package db

// Path returns the path to a user resource, used by the archive connect
// service.
func (u User) Path() string {
	return "users/" + u.Name
}
