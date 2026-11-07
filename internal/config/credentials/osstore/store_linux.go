// Purpose: Store encryption keys through cancellable Linux Secret Service calls on a private bus.
// Pattern: Native OS credential adapter.
// Usage: Vault supplies a fixed service; each operation closes its session and bus connection.
// Related: internal/config/credentials/osstore/secret_connection_linux.go, internal/config/credentials/osstore/secret_prompt_linux.go.
package osstore

import (
	"context"
	"fmt"
	dbus "github.com/godbus/dbus/v5"
	"strings"
	"time"
)

type Store struct{ Service string }

const secretName = "org.freedesktop.secrets"
const secretPath = dbus.ObjectPath("/org/freedesktop/secrets")
const secretInterface = "org.freedesktop.Secret."

type secretClient struct {
	ctx     context.Context
	conn    *dbus.Conn
	session dbus.ObjectPath
}

type secretValue struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func (s Store) open(ctx context.Context) (*secretClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.Service == "" || len(s.Service) > 128 || strings.ContainsAny(s.Service, "\x00\r\n") {
		return nil, fmt.Errorf("invalid OS credential service")
	}
	conn, err := connectSecretBus(ctx)
	if err != nil {
		return nil, err
	}
	return &secretClient{ctx: ctx, conn: conn}, nil
}

func (c *secretClient) call(path dbus.ObjectPath, method string, args ...any) *dbus.Call {
	return c.conn.Object(secretName, path).CallWithContext(c.ctx, secretInterface+method, 0, args...)
}

func (c *secretClient) close() {
	if c.session != "" && c.ctx.Err() == nil {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second)
		_ = c.conn.Object(secretName, c.session).CallWithContext(ctx, secretInterface+"Session.Close", 0).Err
		cancel()
	}
	_ = c.conn.Close()
}

func (c *secretClient) startSession() error {
	var ignored dbus.Variant
	if err := c.call(secretPath, "Service.OpenSession", "plain", dbus.MakeVariant("")).Store(&ignored, &c.session); err != nil {
		return err
	}
	if c.session == "/" || !c.session.IsValid() {
		return fmt.Errorf("invalid Secret Service session")
	}
	return nil
}

func (c *secretClient) search(attributes map[string]string) ([]dbus.ObjectPath, error) {
	var unlocked, locked []dbus.ObjectPath
	if err := c.call(secretPath, "Service.SearchItems", attributes).Store(&unlocked, &locked); err != nil {
		return nil, err
	}
	if len(locked) > 0 {
		if err := c.unlock(locked); err != nil {
			return nil, err
		}
	}
	return append(unlocked, locked...), nil
}

func (s Store) Set(ctx context.Context, ref, value string) error {
	c, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	var collections dbus.Variant
	if err := c.conn.Object(secretName, secretPath).CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, secretInterface+"Service", "Collections").Store(&collections); err != nil {
		return err
	}
	paths, ok := collections.Value().([]dbus.ObjectPath)
	if !ok {
		return fmt.Errorf("invalid Secret Service collections")
	}
	collection := dbus.ObjectPath("/")
	for _, path := range paths {
		if path == "/org/freedesktop/secrets/collection/login" {
			collection = path
			break
		}
	}
	if collection == "/" {
		if err := c.call(secretPath, "Service.ReadAlias", "default").Store(&collection); err != nil {
			return err
		}
	}
	if collection == "/" || !collection.IsValid() {
		return fmt.Errorf("configure a default Secret Service collection")
	}
	if err := c.unlock([]dbus.ObjectPath{collection}); err != nil {
		return err
	}
	if err := c.startSession(); err != nil {
		return err
	}
	properties := map[string]dbus.Variant{
		secretInterface + "Item.Label":      dbus.MakeVariant("MECA encryption key"),
		secretInterface + "Item.Attributes": dbus.MakeVariant(map[string]string{"service": s.Service, "username": ref}),
	}
	secret := secretValue{c.session, []byte{}, []byte(value), "text/plain; charset=utf8"}
	defer clear(secret.Value)
	var item, prompt dbus.ObjectPath
	if err := c.call(collection, "Collection.CreateItem", properties, secret, true).Store(&item, &prompt); err != nil {
		return err
	}
	return c.prompt(prompt)
}

func (s Store) Get(ctx context.Context, ref string) (string, error) {
	c, err := s.open(ctx)
	if err != nil {
		return "", err
	}
	defer c.close()
	items, err := c.search(map[string]string{"service": s.Service, "username": ref})
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", ErrNotFound
	}
	if len(items) != 1 {
		return "", fmt.Errorf("ambiguous OS encryption key; preserve the OS store")
	}
	if err := c.startSession(); err != nil {
		return "", err
	}
	var value secretValue
	if err := c.call(items[0], "Item.GetSecret", c.session).Store(&value); err != nil {
		return "", err
	}
	defer clear(value.Value)
	if value.Session != c.session || len(value.Parameters) != 0 {
		return "", fmt.Errorf("invalid Secret Service response")
	}
	return string(value.Value), nil
}

func (s Store) Clear(ctx context.Context) error {
	c, err := s.open(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	items, err := c.search(map[string]string{"service": s.Service})
	if err != nil {
		return err
	}
	for _, item := range items {
		var prompt dbus.ObjectPath
		if err := c.call(item, "Item.Delete").Store(&prompt); err != nil {
			return err
		}
		if err := c.prompt(prompt); err != nil {
			return err
		}
	}
	return nil
}
