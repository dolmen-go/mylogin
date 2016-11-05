package mylogin

// Section represents one section of the plaintext content of mylogin.cnf.
type Section struct {
	Name  string `json:"name"`
	Login Login  `json:"login"`
}

// Sections represents the structured content on the plaintext of mylogin.cnf.
type Sections []Section

// Login returns the Login from the section with the given name.
func (sections Sections) Login(section string) *Login {
	for _, s := range sections {
		if s.Name == section {
			return &s.Login
		}
	}
	return nil
}

// Merge returns a single Login which is the result of the ordered merge
// of the section with the given names (see Login.Merge).
// For each option the last section that has a value has the precedence.
func (sections Sections) Merge(sectionNames []string) (login *Login) {
	for _, s := range sectionNames {
		if s == "" {
			s = DefaultSection
		}
		l := sections.Login(s)
		if l.IsEmpty() {
			continue
		}
		if login == nil {
			login = new(Login)
		}
		login.Merge(l)
	}

	return
}
