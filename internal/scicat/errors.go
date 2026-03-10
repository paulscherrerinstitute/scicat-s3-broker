package scicat

import "fmt"

type DatasetNotAccessibleError struct {
	Pid string
}

type DatasetNotFoundError struct {
	Pid string
}

func (e DatasetNotAccessibleError) Error() string {
	return fmt.Sprintf("Dataset %s not accessible", e.Pid)
}
func (e DatasetNotFoundError) Error() string {
	return fmt.Sprintf("Dataset with PID %s not found.", e.Pid)
}

type PublishedDataNotFoundError struct {
	Id string
}

func (e PublishedDataNotFoundError) Error() string {
	return fmt.Sprintf("No published data found with id %s", e.Id)
}
