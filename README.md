# handlehttp

Generic Go http handler that handles requests and sends responses and validation problems based on your structs.

Inspired by [Composable HTTP Handlers using generics by Willem Schots](https://www.willem.dev/articles/generic-http-handlers/).

## usage

to use pointer receivers, the json body needs to have an empty object, otherwise initialisation fails.
