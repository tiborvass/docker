// Package namespace is a package used to manage a namespace configuration
// and resolve names into named repositories.
package namespace

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

var (
	// ErrResolutionFail is used when a namespace could not be found for a
	// scope.
	ErrResolutionFail = errors.New("resolution failed: could not find namespace configuration")
)

// RemoteEndpoint represents a remote server which serves
// an API for specific functions (i.e. push, pull, search, trust).
type RemoteEndpoint struct {
	// Action represents the functional API that this remote
	// endpoint is serving.  Their may be multiple actions
	// allowed for a single API, but an action should always
	// map to exactly one API. A subset of the API may be used
	// for specific actions.
	Action string

	// BaseURL represents the URL in which the API is served.
	// For HTTP APIs this may include a path which should be
	// considered the root for the API. The information needed
	// to interpret how this base URL is used may be derived
	// from the flags for this endpoint.
	BaseURL *url.URL

	// Priority represents the relative priority of this
	// endpoint over other endpoints with the same action.
	// This priority defaults to 0 and endpoints with a
	// higher priority should be considered better matches
	// over endpoints with a lower priority.
	Priority int

	// Flags holds action-specific flags for the endpoint
	// which include version information and specific
	// requirements for interacting with the endpoint.
	Flags map[string]string
}

var (
	flagExp     = regexp.MustCompile("^([[:alnum:]][[:word:]]*)(?:=([[:graph:]]+))$")
	priorityExp = regexp.MustCompile("^[[:digit:]]+$")
)

// createEndpoint parses the entry into a RemoteEndpoint.
// The arguments are treated as <endpoint> [<priority>] [<flag>[=<value>]]...
func createEndpoint(entry Entry) (*RemoteEndpoint, error) {
	if len(entry.args) == 0 {
		return nil, errors.New("missing endpoint argument")
	}
	base, err := url.Parse(entry.args[0])
	if err != nil {
		return nil, err
	}
	// If scheme is empty, reparse with default scheme
	if base.Scheme == "" {
		base, err = url.Parse("https://" + entry.args[0])
		if err != nil {
			return nil, err
		}
	}

	priority := 0
	var prioritySet bool
	flags := map[string]string{}
	for _, arg := range entry.args[1:] {
		if !prioritySet {
			// Attempt match on
			if priorityExp.MatchString(arg) {
				i, err := strconv.Atoi(arg)
				if err != nil {
					return nil, err
				}
				priority = i
				prioritySet = true
				continue
			}
		}
		matches := flagExp.FindStringSubmatch(arg)
		if len(matches) == 2 {
			flags[matches[1]] = ""
		} else if len(matches) == 3 {
			flags[matches[1]] = matches[2]
		} else {
			return nil, fmt.Errorf("invalid flag %q", arg)
		}
	}

	return &RemoteEndpoint{
		Action:   entry.action,
		BaseURL:  base,
		Priority: priority,
		Flags:    flags,
	}, nil
}

// Resolver resolves a fully qualified name into
// a namespace configuration.
type Resolver interface {
	Resolve(name string) (*Entries, error)
}

// MultiResolver does resolution across multiple resolvers
// in order of precedence. The next resolver is only used if
// a resolver returns no entries or there is a namespace entry
// with the targeted namespace matching the name of the scope
// being looked up.
type MultiResolver struct {
	resolvers []Resolver
}

// NewMultiResolver returns a new resolver which attempts
// to use multiple resolvers in order to resolve a name
// to a set of entries. Resolution happens in the order
// the resolvers are passed, first having higher
// precendence.
func NewMultiResolver(resolver ...Resolver) Resolver {
	return &MultiResolver{
		resolvers: resolver,
	}
}

func recursiveResolve(es *Entries, name string, resolvers []Resolver) (*Entries, error) {
	if len(resolvers) == 0 {
		return es, nil
	}
	resolved, err := resolvers[0].Resolve(name)
	if err != nil {
		return nil, err
	}
	if resolved != nil && len(resolved.entries) > 0 {
		for _, entry := range resolved.entries {
			// Recurse on namespace actions
			if entry.action == actionNamespace {
				for _, arg := range entry.args {
					sub, err := recursiveResolve(resolved, arg, resolvers[1:])
					if err != nil {
						return nil, err
					}
					resolved, err = resolved.Join(sub)
					if err != nil {
						return nil, err
					}
				}
			}
		}

		return es.Join(resolved)
	}

	return recursiveResolve(es, name, resolvers[1:])
}

// Resolve resolves a name into a list of entries using multiple
// resolvers to collect the list. The resolved list is guaranteed
// to be unique even if multiple resolvers are called.
func (mr *MultiResolver) Resolve(name string) (*Entries, error) {
	return recursiveResolve(NewEntries(), name, mr.resolvers)
}

// SimpleResolver is a resolver which uses a static set of entries
// to resolve a names based on the entry scope
type SimpleResolver struct {
	entries     map[scope]*Entries
	prefixMatch bool
}

func (sr *SimpleResolver) resolveEntries(es *Entries, name string) error {
	entries := sr.entries[scope(name)]
	if entries != nil {
		var extended []string
		for _, entry := range entries.entries {
			if err := es.Add(entry); err != nil {
				return err
			}
			if entry.action == actionNamespace {
				for _, arg := range entry.args {
					// When arg is not the name, also use additional scope
					if arg != name {
						scope, err := parseScope(arg)
						if err != nil {
							return err
						}
						if !scope.Contains(name) {
							return errors.New("invalid extension: must extend ancestor scope")
						}
						extended = append(extended, arg)
					}
				}
			}
		}
		for _, extend := range extended {
			if err := sr.resolveEntries(es, extend); err != nil {
				return err
			}
		}
	}

	// No results produced, fallback to any prefix matches
	if sr.prefixMatch && len(es.entries) == 0 {
		for s, entries := range sr.entries {
			if !s.Contains(name) {
				continue
			}
			for _, entry := range entries.entries {
				if err := es.Add(entry); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Resolve resolves a name into a list of entries based on a static
// set of entries.
func (sr *SimpleResolver) Resolve(name string) (*Entries, error) {
	entries := NewEntries()
	if err := sr.resolveEntries(entries, name); err != nil {
		return nil, err
	}
	return entries, nil
}

// NewSimpleResolver returns a resolver which will only match from
// the provided set of entries. If prefixMatch is set to true, then
// the resolver will return entries which have a prefix match if
// no other entries were found.
func NewSimpleResolver(base *Entries, prefixMatch bool) Resolver {
	entries := map[scope]*Entries{}
	for _, entry := range base.entries {
		scoped, ok := entries[entry.scope]
		if !ok {
			scoped = NewEntries()
			entries[entry.scope] = scoped
		}
		scoped.Add(entry)
	}
	return &SimpleResolver{
		entries:     entries,
		prefixMatch: prefixMatch,
	}
}

// ExtendResolver extends the set of resolved entries only if
// entries were found.
type ExtendResolver struct {
	extendResolver Resolver
	baseResolver   Resolver
}

// Resolve resolves a name into a list of entries extending a
// list from another Resolver with a static set of entries.
func (er *ExtendResolver) Resolve(name string) (*Entries, error) {
	entries, err := er.baseResolver.Resolve(name)
	if err != nil {
		return nil, err
	}
	if len(entries.entries) > 0 {
		extended, err := er.extendResolver.Resolve(name)
		if err != nil {
			return nil, err
		}
		return entries.Join(extended)
	}
	return entries, nil
}

// NewExtendResolver returns a new Resolver which will extended the
// entries found through the given resolver with the given
// extended entries.
func NewExtendResolver(extension *Entries, resolver Resolver) Resolver {
	simple := NewSimpleResolver(extension, false)
	return &ExtendResolver{
		extendResolver: simple,
		baseResolver:   resolver,
	}
}

// NewDefaultFileResolver returns a new MultiResolver which uses
// the entries from the given file as the highest precendent then
// the following resolvers in the order they are given.
func NewDefaultFileResolver(namespaceFile string, resolvers ...Resolver) (Resolver, error) {
	// Read base entries from f.NamespaceFile
	nsf, err := os.Open(namespaceFile)
	if err != nil {
		return nil, err
	}

	entries, err := ReadEntries(nsf)
	if err != nil {
		return nil, err
	}
	resolvers = append([]Resolver{NewSimpleResolver(entries, true)}, resolvers...)
	return NewMultiResolver(resolvers...), nil

}

// GetRemoteEndpoints returns a list of remote endpoints from
// the set of entries.
func GetRemoteEndpoints(entries *Entries) ([]*RemoteEndpoint, error) {
	endpoints := []*RemoteEndpoint{}
	for _, entry := range entries.entries {
		switch entry.action {
		case actionIndex:
			fallthrough
		case actionTrust:
			fallthrough
		case actionPull:
			fallthrough
		case actionPush:
			endpoint, err := createEndpoint(entry)
			endpoints = append(endpoints, endpoint)
			if err != nil {
				return nil, err
			}
		}
	}
	return endpoints, nil
}
