package scicat

import "fmt"

type DatasetNotAccessibleError struct {
	Pid string
}

type NoUrlsAvailableError struct {
	Pid string
}

func (e DatasetNotAccessibleError) Error() string {
	return fmt.Sprintf("Dataset %s not accessible", e.Pid)
}
func (e NoUrlsAvailableError) Error() string {
	return fmt.Sprintf("No URLs available for %s. Trigger a URL retrieve job in SciCat", e.Pid)
}

type PublishedDataNotFoundError struct {
	Id string
}

func (e PublishedDataNotFoundError) Error() string {
	return fmt.Sprintf("No published data found with id %s", e.Id)
}
