package server

// Server is the initial composition root placeholder. Network transports and
// storage adapters are attached in later waves; construction already enforces
// configuration validity so bootstrap behavior remains explicit.
type Server struct {
	config Config
}

func New(config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Server{config: config}, nil
}

func (s *Server) Config() Config {
	if s == nil {
		return Config{}
	}
	return s.config
}
