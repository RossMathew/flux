package policy

import (
	"encoding/json"
	"strings"

	"github.com/weaveworks/flux"
)

const (
	Ignore     = Policy("ignore")
	Locked     = Policy("locked")
	LockedUser = Policy("locked_user")
	LockedMsg  = Policy("locked_msg")
	Automated  = Policy("automated")
	TagAll     = Policy("tag_all")
)

// Policy is an string, denoting the current deployment policy of a service,
// e.g. automated, or locked.
type Policy string

func Boolean(policy Policy) bool {
	switch policy {
	case Locked, Automated, Ignore:
		return true
	}
	return false
}

func TagPrefix(container string) Policy {
	return Policy("tag." + container)
}

func Tag(policy Policy) bool {
	return strings.HasPrefix(string(policy), "tag.")
}

func GetTagPattern(services ResourceMap, service flux.ResourceID, container string) string {
	if services == nil {
		return "*"
	}
	policies := services[service]
	if pattern, ok := policies.Get(TagPrefix(container)); ok {
		return strings.TrimPrefix(pattern, "glob:")
	}
	return "*"
}

type Updates map[flux.ResourceID]Update

type Update struct {
	Add    Set `json:"add"`
	Remove Set `json:"remove"`
}

type Set map[Policy]string

// We used to specify a set of policies as []Policy, and in some places
// it may be so serialised.
func (s *Set) UnmarshalJSON(in []byte) error {
	type set Set
	if err := json.Unmarshal(in, (*set)(s)); err != nil {
		var list []Policy
		if err = json.Unmarshal(in, &list); err != nil {
			return err
		}
		var s1 = Set{}
		*s = s1.Add(list...)
	}
	return nil
}

func (s Set) String() string {
	var ps []string
	for p, v := range s {
		ps = append(ps, string(p)+":"+v)
	}
	return "{" + strings.Join(ps, ", ") + "}"
}

func (s Set) Add(ps ...Policy) Set {
	s = clone(s)
	for _, p := range ps {
		s[p] = "true"
	}
	return s
}

func (s Set) Set(p Policy, v string) Set {
	s = clone(s)
	s[p] = v
	return s
}

func clone(s Set) Set {
	newMap := Set{}
	for p, v := range s {
		newMap[p] = v
	}
	return newMap
}

// Has returns true if a resource has a particular policy present, and
// for boolean policies, if it is set to true.
func (s Set) Has(needle Policy) bool {
	for p, v := range s {
		if p == needle {
			if Boolean(needle) {
				return v == "true"
			}
			return true
		}
	}
	return false
}

func (s Set) Get(p Policy) (string, bool) {
	v, ok := s[p]
	return v, ok
}

func (s Set) Without(omit Policy) Set {
	newMap := Set{}
	for p, v := range s {
		if p != omit {
			newMap[p] = v
		}
	}
	return newMap
}

func (s Set) ToStringMap() map[string]string {
	m := map[string]string{}
	for p, v := range s {
		m[string(p)] = v
	}
	return m
}

type ResourceMap map[flux.ResourceID]Set

func (s ResourceMap) ToSlice() []flux.ResourceID {
	slice := []flux.ResourceID{}
	for service, _ := range s {
		slice = append(slice, service)
	}
	return slice
}

func (s ResourceMap) Contains(id flux.ResourceID) bool {
	_, ok := s[id]
	return ok
}

func (s ResourceMap) Without(other ResourceMap) ResourceMap {
	newMap := ResourceMap{}
	for k, v := range s {
		if !other.Contains(k) {
			newMap[k] = v
		}
	}
	return newMap
}

func (s ResourceMap) OnlyWithPolicy(p Policy) ResourceMap {
	newMap := ResourceMap{}
	for k, v := range s {
		if v.Has(p) {
			newMap[k] = v
		}
	}
	return newMap
}
