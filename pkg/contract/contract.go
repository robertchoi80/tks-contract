package contract

import (
	"strconv"
	"time"

	pb "github.com/sktelecom/tks-proto/pbgo"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Contract represents a contract data
type Contract struct {
	ContractorName    string            `json:"contractorName"`
	ID                ID                `json:"contractID"`
	AvailableServices []string          `json:"availableServices,omitempty"`
	CspID             ID                `json:"cspId,omitempty"`
	LastUpdatedTs     *LastUpdatedTime  `json:"lastUpdatedTs"`
	Quota             *pb.ContractQuota `json:"quota"`
}

// ID is a global unique ID for the contracts.
// ID is generated by CBP or external site.
type ID string

// LastUpdatedTime renamed time.Time type.
type LastUpdatedTime struct {
	time.Time
}

// UnmarshalJSON parses the Unix time and stores the result in ts
func (ts *LastUpdatedTime) UnmarshalJSON(data []byte) error {
	unix, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	ts.Time = time.Unix(unix, 0)
	return nil
}

// Timestamppb converts time.Time type to timestamppb.Timestamp.
func (ts LastUpdatedTime) Timestamppb() *timestamppb.Timestamp {
	return timestamppb.New(ts.Time)
}
