package response

import "github.com/tedwangl/go-util/pkg/model/example"


type ExaCustomerResponse struct {
	Customer example.ExaCustomer `json:"customer"`
}
