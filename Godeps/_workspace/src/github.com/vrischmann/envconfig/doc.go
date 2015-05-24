/*
Package envconfig implements a configuration reader which reads each value from an environment variable.

The basic idea is that you define a configuration struct, like this:

    var conf struct {
        Addr string
        Port int
        Auth struct {
            Key      string
            Endpoint string
        }
        Partitions []int
        Shards     []struct {
            Name string
            Id   int
        }
    }

Once you have that, you need to initialize the configuration:

    if err := envconfig.Init(&conf); err != nil {
        log.Fatalln(err)
    }

Then it's just a matter of setting the environment variables when calling your binary:

    ADDR=localhost PORT=6379 AUTH_KEY=foobar ./mybinary
    
Layout of the conf struct

Your conf struct must follow the following rules:
 - no unexported fields
 - only supported types (no map fields for example)

Naming of the keys

By default, keys must follow a specific naming scheme:
 - all uppercase
 - a single _ to go down a level in the hierarchy of the struct

Examples:
 - ADDR
 - AUTH_ENDPOINT
 - PARTITIONS

You can override the expected name of the key for a single field using a field tag:

    var conf struct {
        Name `envconfig:"customName"`
    }

Now envconfig will read the environment variable named "customName".

Content of the variables

There are three types of content for a single variable:

 - for simple types, a single string representing the value, and parseable into the type.
 - for slices or arrays, a comma-separated list of strings. Each string must be parseable into the element type of the slice or array.
 - for structs, a comma-separated list of specially formatted strings representing structs.

Example of a valid slice value:
    foo,bar,baz

The format for a struct is as follow:
 - prefixed with {
 - suffixed with }
 - contains a comma-separated list of field values, in the order in which they are defined in the struct

Example of a valid struct value:
    type MyStruct struct {
        Name    string
        Id      int
        Timeout time.Duration
    }

    {foobar,10,120s}

Example of a valid slice of struct values:
    {foobar,10,120s},{barbaz,20,50s}

Optional values

Sometimes you don't absolutely need a value. Here's how we tell envconfig a value is optional:

    var conf struct {
        Name string `envconfig:"optional"`
    }

Supported types

 - bool
 - string
 - intX
 - uintX
 - floatX
 - time.Duration
 - pointers to all of the above types

Custom unmarshaler

When the standard types are not enough, you will want to use a custom unmarshaler for your types.

You do this by implementing Unmarshaler on your type. Here's an example:

    type connectionType uint

    const (
        tlsConnection connectionType = iota
        insecureConnection
    )

    func (t *connectionType) Unmarshal(s string) error {
        switch s {
            case "tls":
                *t = tlsConnection
            case "insecure":
                *t = insecureConnection
            default:
                return fmt.Errorf("unable to unmarshal %s to a connection type", s)
        }

        return nil
    }

*/
package envconfig
