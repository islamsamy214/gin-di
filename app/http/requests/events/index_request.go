/*
 * Package events holds the form requests for the events endpoints, mirroring
 * Laravel's app/Http/Requests.
 */
package events

// Pagination defaults, applied by Defaults when the caller omits them.
const (
	DefaultPage    = 1
	DefaultPerPage = 10
)

/*
 * IndexRequest is the query string for GET /events.
 *
 * PerPage is capped in the tag: an uncapped page size lets a single request
 * select the entire table, which is a denial-of-service vector dressed up as a
 * feature. The cap and the default are separate concerns — the tag rejects
 * out-of-range values, Defaults fills in absent ones.
 */
type IndexRequest struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

/*
 * Defaults fills in the values the caller left out.
 *
 * The equivalent of a Laravel form request's prepareForValidation, except it
 * runs after binding: gin has no pre-bind hook, and omitempty means an absent
 * value arrives here as a zero rather than as a validation failure.
 */
func (request *IndexRequest) Defaults() {
	if request.Page < 1 {
		request.Page = DefaultPage
	}

	if request.PerPage < 1 {
		request.PerPage = DefaultPerPage
	}
}
