package main

type OmdbService struct {
	omdbClient *OMDBClient
}

func NewOmdbService() (*OmdbService, error) {
	client, err := NewOMDBClient()
	if err != nil {
		return nil, err
	}

	return &OmdbService{
		omdbClient: client,
	}, nil
}

func MediaSearch(s *OmdbService, title string, year *int) (*MovieResponse, error) {
	result, err := s.omdbClient.GetMovieByTitle(title)
	if err != nil {
		return nil, err
	}

	return result, nil
}
