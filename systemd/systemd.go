package systemd

import (
	"context"
	"time"

	"github.com/godbus/dbus"
)

// systemd DBus api wrapper

const (
	destination = "org.freedesktop.systemd1"
	objectPath  = "/org/freedesktop/systemd1"
)

// Conn represents a systemd dbus connection
type Conn struct {
	obj dbus.BusObject
}

// Connect establishes a dbus SystemBus connection
func Connect() (*Conn, error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	return &Conn{
		obj: bus.Object(destination, objectPath),
	}, nil
}

// call calls a dbus method on the systemd object
func (c *Conn) call(parentContext context.Context, method string, args ...interface{}) (*dbus.Call, error) {
	ctx, cancel := context.WithTimeout(parentContext, time.Millisecond*300)
	defer cancel()
	ch := make(chan *dbus.Call, 1)
	c.obj.Go(method, 0, ch, args...)
	select {
	case <-ctx.Done():
		// goroutine might leak on timeout, but we can't do anything about it
		return nil, ctx.Err()
	case call := <-ch:
		if call.Err != nil {
			return nil, call.Err
		}
		return call, nil
	}
}

// StartUnit starts a sytemd unit
func (c *Conn) StartUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.StartUnit", unitName, "replace")
	return err
}

// RestartUnit restarts a systemd unit
func (c *Conn) RestartUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.RestartUnit", unitName, "replace")
	return err
}

// ReloadUnit reloads a systemd unit
func (c *Conn) ReloadUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.ReloadUnit", unitName, "replace")
	return err
}

// StopUnit stops a systemd unit
func (c *Conn) StopUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.StopUnit", unitName, "replace")
	return err
}

// EnableUnit enables a systemd unit
func (c *Conn) EnableUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.EnableUnitFiles", []string{unitName}, false, false)
	return err
}

// DisableUnit disables a systemd unit
func (c *Conn) DisableUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "org.freedesktop.systemd1.Manager.DisableUnitFiles", []string{unitName}, false)
	return err
}

// ListUnits returns a list of systemd units
type Unit struct {
	Name        string
	Description string
	LoadState   string
	ActiveState string
}

func (c *Conn) ListUnits(ctx context.Context) ([]Unit, error) {
	type listresult struct {
		Name        string
		Description string
		LoadState   string
		ActiveState string
		SubState    string
		Following   string
		ObjectPath  string
		JobId       uint
		Jobtype     string
		JobPath     string
	}
	var out []listresult
	call, err := c.call(ctx, "org.freedesktop.systemd1.Manager.ListUnits")
	if err != nil {
		return nil, err
	}
	err = call.Store(&out)
	if err != nil {
		return nil, err
	}
	var units []Unit
	for _, result := range out {
		units = append(units, Unit{
			Name:        result.Name,
			Description: result.Description,
			LoadState:   result.LoadState,
			ActiveState: result.ActiveState,
		})
	}
	return units, nil
}
