package systemd

import (
	"context"
	"log"
	"time"

	"github.com/godbus/dbus"
)

// systemd DBus api wrapper

const (
	destination = "org.freedesktop.systemd1"
	objectPath  = "/org/freedesktop/systemd1"
)

type Conn struct {
	obj dbus.BusObject
}

func Connect() (*Conn, error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	return &Conn{
		obj: bus.Object(destination, objectPath),
	}, nil
}

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

func (c *Conn) StartUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "StartUnit", unitName, "replace")
	return err
}

func (c *Conn) RestartUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "RestartUnit", unitName, "replace")
	return err
}

func (c *Conn) ReloadUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "ReloadUnit", unitName, "replace")
	return err
}

func (c *Conn) StopUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "StopUnit", unitName, "replace")
	return err
}

func (c *Conn) EnableUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "EnableUnitFiles", []string{unitName}, false, false)
	return err
}

func (c *Conn) DisableUnit(ctx context.Context, unitName string) error {
	_, err := c.call(ctx, "DisableUnitFiles", []string{unitName}, false)
	return err
}

func (c *Conn) ListUnits(ctx context.Context, unitName string) error {
	var out []interface{}
	call, err := c.call(ctx, "StopUnit", unitName, "replace")
	if err != nil {
		return err
	}
	err = call.Store(out)
	if err != nil {
		return err
	}
	log.Println("got units", out)
	return nil
}
