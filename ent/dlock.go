// Code generated by ent, DO NOT EDIT.

package ent

import (
	"github.com/itsabgr/dblock/ent/dlock"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

// DLock is the model entity for the DLock schema.
type DLock struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Holder holds the value of the "holder" field.
	Holder uuid.UUID `json:"holder,omitempty"`
	// Deadline holds the value of the "deadline" field.
	Deadline     int64 `json:"deadline,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*DLock) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case dlock.FieldDeadline:
			values[i] = new(sql.NullInt64)
		case dlock.FieldID:
			values[i] = new(sql.NullString)
		case dlock.FieldHolder:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the DLock fields.
func (d *DLock) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case dlock.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				d.ID = value.String
			}
		case dlock.FieldHolder:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field holder", values[i])
			} else if value != nil {
				d.Holder = *value
			}
		case dlock.FieldDeadline:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field deadline", values[i])
			} else if value.Valid {
				d.Deadline = value.Int64
			}
		default:
			d.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the DLock.
// This includes values selected through modifiers, order, etc.
func (d *DLock) Value(name string) (ent.Value, error) {
	return d.selectValues.Get(name)
}

// Update returns a builder for updating this DLock.
// Note that you need to call DLock.Unwrap() before calling this method if this DLock
// was returned from a transaction, and the transaction was committed or rolled back.
func (d *DLock) Update() *DLockUpdateOne {
	return NewDLockClient(d.config).UpdateOne(d)
}

// Unwrap unwraps the DLock entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (d *DLock) Unwrap() *DLock {
	_tx, ok := d.config.driver.(*txDriver)
	if !ok {
		panic("ent: DLock is not a transactional entity")
	}
	d.config.driver = _tx.drv
	return d
}

// String implements the fmt.Stringer.
func (d *DLock) String() string {
	var builder strings.Builder
	builder.WriteString("DLock(")
	builder.WriteString(fmt.Sprintf("id=%v, ", d.ID))
	builder.WriteString("holder=")
	builder.WriteString(fmt.Sprintf("%v", d.Holder))
	builder.WriteString(", ")
	builder.WriteString("deadline=")
	builder.WriteString(fmt.Sprintf("%v", d.Deadline))
	builder.WriteByte(')')
	return builder.String()
}

// DLocks is a parsable slice of DLock.
type DLocks []*DLock
