package pkg

import (
	"fmt"

	"github.com/typstify/tpix-cli/api"
)

// ListPkgs queries packages/templates from TPIX package index, this includes
// public and private namespaces the user have access permissions.
func (s *TypstPkgService) SearchPkgs(namespace string, kind string, category string, query string) ([]TypstPkg, int, error) {
	if s.searchState.searchKey != "" && s.searchState.searchKey == fmt.Sprintf("%s:%s:%s", namespace, kind, query) {
		cachedData := s.searchState.data.Load()
		if cachedData != nil {
			pkgs := convertToPkg(cachedData)
			return pkgs, cachedData.Total, nil
		}
	}

	if s.searchState.loading.Load() {
		return nil, 0, nil
	}

	resp, err := s.search(namespace, kind, category, query)
	if err != nil {
		return nil, 0, err
	}

	pkgs := convertToPkg(resp)
	s.searchState.data.Store(resp)

	return pkgs, resp.Total, nil
}

func (s *TypstPkgService) search(namespace string, kind string, category string, query string) (*api.SearchResponse, error) {
	if !s.searchState.loading.CompareAndSwap(false, true) {
		return nil, nil
	}

	defer func() {
		s.searchState.loading.CompareAndSwap(true, false)
	}()

	return s.tpixClient.SearchPackages(namespace, query, kind, category, "", 100)

}

func convertToPkg(resp *api.SearchResponse) []TypstPkg {
	pkgs := make([]TypstPkg, len(resp.Results))
	for i, result := range resp.Results {
		pkgs[i] = TypstPkg{
			SearchResult: result,
		}
	}

	return pkgs
}
