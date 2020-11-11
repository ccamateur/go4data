// Package workflow has failures that ca nbe sent by Processors to either logging or just stdout etc
package workflow

import (
	"errors"
	"fmt"

	"github.com/perbol/workflow/payload"
)

var (
	//ErrIngressRelationshipNeeded is when a processor isn't getting the needed ingress
	ErrIngressRelationshipNeeded = errors.New("the processor needs an ingress to properly run")
)

// Failure is the Workflows Custom error handeling struct
// It contains Error and some meta data about what Processor that triggerd the error
type Failure struct {
	// Err is the error that occured
	Err error `json:"error"`
	// Payload is the payload that was being processed when a Failure occured
	Payload payload.Payload `json:"payload"`
	// Processor is the UUID of the procesor that triggers the Error
	Processor uint `json:"processor"`
}

// PrintFailure is a FailureHandler
func PrintFailure(f Failure) {
	if f.Payload == nil {
		fmt.Printf("%s cast by %d with no payload attached\n", f.Err, f.Processor)
		return
	}
	fmt.Printf("%s cast by %d with payload %s\n", f.Err, f.Processor, f.Payload.GetPayload())
}
