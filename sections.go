package mylogin

type Section struct {
	Name  string `json:"name"`
	Login Login  `json:"login"`
}

type Sections []Section

func (sections Sections) Login(section string) *Login {
	for _, s := range sections {
		if s.Name == section {
			return &s.Login
		}
	}
	return nil
}
