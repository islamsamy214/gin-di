package events

/*
 * StoreRequest is the bind target for POST /events.
 *
 * Note what is absent: there is no UserID field. Ownership is taken from the
 * authenticated identity in the request context, never from the body, so a
 * caller cannot create an event owned by somebody else. Binding straight onto
 * models.Event made UserID mass-assignable, which is the underlying reason the
 * controller ended up setting it by hand.
 *
 * The eventdate rule is registered by ValidationServiceProvider. `required`
 * alone accepts any non-empty string, which then reaches Postgres and comes back
 * as a driver error at HTTP 500 — a validation failure has to be caught here.
 */
type StoreRequest struct {
	Name string `json:"name" binding:"required,max=255"`
	Date string `json:"date" binding:"required,eventdate"`
}
