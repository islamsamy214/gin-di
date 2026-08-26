package resources

/*
 * Meta describes the page a collection was taken from.
 *
 * Count is the size of this page, not the total row count. Reporting a true
 * total needs a second COUNT(*) query per request, which is not worth paying for
 * until a client actually needs to render page numbers.
 */
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Count   int `json:"count"`
}

// NewMeta describes a page given the request's pagination and the rows returned.
func NewMeta(page, perPage, count int) Meta {
	return Meta{Page: page, PerPage: perPage, Count: count}
}

/*
 * EventCollection is the payload of a paginated events response.
 *
 * The metadata sits alongside the rows inside the envelope's data key rather
 * than beside it at the top level, so the envelope itself stays exactly four
 * keys wide regardless of what any endpoint returns.
 *
 * A named type rather than an inline map, so the wire shape is stated once here
 * instead of being re-spelled in every handler that returns a page of events.
 */
type EventCollection struct {
	Events []EventResource `json:"events"`
	Meta   Meta            `json:"meta"`
}

// NewEventPage assembles a page of events with its metadata.
func NewEventPage(events []EventResource, page, perPage int) EventCollection {
	return EventCollection{
		Events: events,
		Meta:   NewMeta(page, perPage, len(events)),
	}
}
