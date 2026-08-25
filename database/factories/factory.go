// Package factories builds model fixtures, mirroring Laravel's database/factories.
package factories

import "fmt"

/*
 * Persistable is all a factory needs from a model in order to save it.
 *
 * Kept to one method so any model satisfies it without extra plumbing.
 */
type Persistable interface {
	Create() error
}

/*
 * Definition builds one instance.
 *
 * The sequence starts at 1 and increments per instance in a batch, which is
 * what lets a definition produce distinct-but-predictable values the way
 * Laravel's sequences do.
 */
type Definition[T Persistable] func(sequence int) (T, error)

/*
 * Factory produces model fixtures for seeders and tests.
 *
 * The API is fluent and chainable in the Eloquent style:
 *
 *	factories.UserFactory().Count(3).Create()
 *	factories.EventFactory().State(func(e *models.Event) { e.UserId = id }).CreateOne()
 */
type Factory[T Persistable] struct {
	definition Definition[T]
	count      int
	states     []func(T)
}

/*
 * New starts a factory from a definition.
 *
 * @return *Factory[T] A factory that makes one instance unless Count says more.
 */
func New[T Persistable](definition Definition[T]) *Factory[T] {
	return &Factory[T]{
		definition: definition,
		count:      1,
	}
}

/*
 * Count sets how many instances to build.
 *
 * @return *Factory[T] The same factory, for chaining.
 */
func (factory *Factory[T]) Count(count int) *Factory[T] {
	factory.count = count

	return factory
}

/*
 * State layers an override onto every instance, like a Laravel factory state.
 *
 * States apply in the order they were added, after the definition has run.
 *
 * @return *Factory[T] The same factory, for chaining.
 */
func (factory *Factory[T]) State(state func(T)) *Factory[T] {
	factory.states = append(factory.states, state)

	return factory
}

/*
 * Make builds the instances without saving them.
 *
 * Needs no database, so tests that only care about the shape of a model can
 * use it directly.
 *
 * @return []T   The built instances.
 * @return error If any definition failed.
 */
func (factory *Factory[T]) Make() ([]T, error) {
	if factory.count < 0 {
		return nil, fmt.Errorf("count must not be negative, got %d", factory.count)
	}

	instances := make([]T, 0, factory.count)

	for sequence := 1; sequence <= factory.count; sequence++ {
		instance, err := factory.definition(sequence)
		if err != nil {
			return nil, fmt.Errorf("building instance %d: %w", sequence, err)
		}

		for _, state := range factory.states {
			state(instance)
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

/*
 * MakeOne builds a single unsaved instance.
 *
 * @return T     The built instance.
 * @return error If the definition failed, or Count is not exactly 1.
 */
func (factory *Factory[T]) MakeOne() (T, error) {
	var zero T

	instances, err := factory.Make()
	if err != nil {
		return zero, err
	}

	if len(instances) != 1 {
		return zero, fmt.Errorf("MakeOne needs a count of 1, got %d", len(instances))
	}

	return instances[0], nil
}

/*
 * Create builds the instances and saves each one.
 *
 * Stops at the first save failure so the caller sees the real cause rather
 * than a pile of downstream errors.
 *
 * @return []T   The saved instances, carrying whatever the insert returned.
 * @return error The first failure, or nil.
 */
func (factory *Factory[T]) Create() ([]T, error) {
	instances, err := factory.Make()
	if err != nil {
		return nil, err
	}

	for index, instance := range instances {
		if err := instance.Create(); err != nil {
			return nil, fmt.Errorf("saving instance %d: %w", index+1, err)
		}
	}

	return instances, nil
}

/*
 * CreateOne builds and saves a single instance.
 *
 * @return T     The saved instance.
 * @return error If building or saving failed.
 */
func (factory *Factory[T]) CreateOne() (T, error) {
	var zero T

	instance, err := factory.MakeOne()
	if err != nil {
		return zero, err
	}

	if err := instance.Create(); err != nil {
		return zero, fmt.Errorf("saving instance: %w", err)
	}

	return instance, nil
}
