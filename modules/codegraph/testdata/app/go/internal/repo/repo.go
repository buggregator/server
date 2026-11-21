package repo

type Repo struct{}

func New() *Repo { return &Repo{} }

func (r *Repo) Persist() error { return r.flush() }
