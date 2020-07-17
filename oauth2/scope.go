package oauth2

import (
	"database/sql/driver"
	"strings"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
)

// Resource is the resource that the OAuth server makes available.
type Resource int

// Permission represents what a scope can do to a resource, this will be
// either Read, Write, or Delete.
type Permission int

// In the database token scopes are stored as a byte array of pairs, where the
// first item in the pair is the resource, and the second is the bitmask of
// permissions. For example the byte array
//
//   2, 3
//
// would be the scope,
//
//   build:read,write
//
// or as code,
//
//  scopeItem{
//    Resource:   Build,
//    Permission: Read|Write,
//  }
type Scope []scopeItem

type scopeItem struct {
	Resource   Resource
	Permission Permission
}

//go:generate stringer -type Permission -linecomment
//go:generate stringer -type Resource -linecomment

const (
	Read   Permission = 1 << iota // read
	Write                         // write
	Delete                        // delete
)

const (
	Build     Resource = 1 + iota // build
	Invite                        // invite
	Image                         // image
	Namespace                     // namespace
	Object                        // object
	Variable                      // variable
	Key                           // key
)

var (
	Permissions = []Permission{
		Read,
		Write,
		Delete,
	}

	Resources = []Resource{
		Build,
		Invite,
		Image,
		Namespace,
		Object,
		Variable,
		Key,
	}

	perms map[string]Permission = map[string]Permission{
		"read":   Read,
		"write":  Write,
		"delete": Delete,
	}

	resources map[string]Resource = map[string]Resource{
		"build":     Build,
		"invite":    Invite,
		"image":     Image,
		"namespace": Namespace,
		"object":    Object,
		"variable":  Variable,
		"key":       Key,
	}
)

// NewScope returns a new Scope for resource permissions to be added to.
func NewScope() Scope {
	return Scope(make([]scopeItem, 0))
}

// Diff returns the a new Scope that is the difference between Scopes a and b.
// You would typically check the length of the returned Scope to see if there
// was a difference like so,
//
//   if len(ScopeDiff(a, b)) > 0 {
//       ...
//   }
func ScopeDiff(a, b Scope) Scope {
	m := make(map[int]struct{})

	for _, item := range a {
		m[int(item.Resource)+int(item.Permission)] = struct{}{}
	}

	diff := Scope(make([]scopeItem, 0))

	for _, item := range b {
		if _, ok := m[int(item.Resource)+int(item.Permission)]; !ok {
			diff = append(diff, item)
		}
	}
	return diff
}

// UnmarshalScope takes a space delimited scope string in the format of
// resource:permission and returns the unmarshalled Scope. An error is returned
// if the given scope string is invalid.
func UnmarshalScope(s string) (Scope, error) {
	m := make(map[Resource]Permission)

	sc := Scope(make([]scopeItem, 0))

	parts := strings.Split(s, " ")

	if len(parts) == 0 {
		return sc, nil
	}

	for _, part := range parts {
		itemParts := strings.Split(part, ":")

		resource, ok := resources[itemParts[0]]

		if !ok {
			return sc, errors.New("unknown resource: " + itemParts[0])
		}

		currPerm := m[resource]

		permParts := strings.Split(itemParts[1], ",")

		for _, p := range permParts {
			perm, ok := perms[p]

			if !ok {
				return sc, errors.New("unknow permission: " + p)
			}

			currPerm |= perm
		}
		m[resource] = currPerm
	}

	for res, perm := range m {
		sc = append(sc, scopeItem{
			Resource:   res,
			Permission: perm,
		})
	}
	return sc, nil
}

func (i scopeItem) bytes() []byte { return []byte{byte(i.Resource), byte(i.Permission)} }

func (i scopeItem) String() string {
	s := i.Resource.String() + ":"
	perms := make([]string, 0, 3)

	for _, mask := range Permissions {
		if i.Permission.Has(mask) {
			perms = append(perms, mask.String())
		}
	}
	return s + strings.Join(perms, ",")
}

func (p Permission) Expand() []Permission {
	expanded := make([]Permission, 0)

	for _, mask := range Permissions {
		if p.Has(mask) {
			expanded = append(expanded, mask)
		}
	}
	return expanded
}

// Determine if the given permission mask exists in the permission.
func (p Permission) Has(mask Permission) bool { return (p & mask) == mask }

// Add adds the given resource and permissions to the current scope. If the
// resource already exists in the current scope then the permissions are
// updated with the new permission.
func (sc *Scope) Add(res Resource, perm Permission) {
	for i, it := range *sc {
		if it.Resource == res {
			if it.Permission != perm {
				(*sc)[i].Permission |= perm
				return
			}
		}
	}

	(*sc) = append((*sc), scopeItem{
		Resource:   res,
		Permission: perm,
	})
}

// Scan scans the underlying byte slice value of the given interface into the
// curent Scope if it is valid.
func (sc *Scope) Scan(val interface{}) error {
	if (*sc) == nil {
		(*sc) = Scope(make([]scopeItem, 0))
	}

	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b)%2 != 0 {
		return errors.New("invalid scope bytes")
	}

	i := 0

	for i != len(b) {
		(*sc) = append((*sc), scopeItem{
			Resource:   Resource(b[i]),
			Permission: Permission(b[i+1]),
		})
		i += 2
	}
	return nil
}

// Spread returns a slice of resource:permission strings.
func (sc *Scope) Spread() []string {
	s := make([]string, 0)

	for _, item := range *sc {
		for _, p := range Permissions {
			if item.Permission.Has(p) {
				s = append(s, item.Resource.String()+":"+p.String())
			}
		}
	}
	return s
}

// String returns a space delimited string of all the resources and their
// respective permissions as a single string, for example,
//
//   build:read,write namespace:read variable:read,write,delete
func (sc *Scope) String() string {
	items := make([]string, 0, len((*sc)))

	for _, item := range *sc {
		items = append(items, item.String())
	}
	return strings.Join(items, " ")
}

// UnmarshalText unmarshals the given byte slice into the current Scope
// if it is valid. Under the hood this calls UnmarshalScope, so the byte
// slice is expected to be a space delimited string.
func (sc *Scope) UnmarshalText(b []byte) error {
	var err error
	(*sc), err = UnmarshalScope(string(b))
	return errors.Err(err)
}

// Value returns the raw byte pairs of a Scope that can be inserted into the
// database.
func (sc Scope) Value() (driver.Value, error) {
	b := make([]byte, 0)

	for _, item := range sc {
		b = append(b, item.bytes()...)
	}
	return b, nil
}
