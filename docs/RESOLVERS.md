# Resolver Strategy

The client accepts one or more DNS resolvers. For real DNS transport this has a
subtle constraint: all fragments of one QUIC packet must go through the same
resolver.

If fragments of a single packet are spread across multiple recursive resolvers,
the Slipstream server can see them as different UDP peers and fail to
reassemble the original QUIC packet. The client therefore chooses one resolver
per QUIC packet and sends all DNS fragments for that packet to the chosen
resolver.

Different packets still rotate across the configured resolver list. QUIC
retransmission can therefore move later packets to another resolver when one
path is unhealthy.

Resolver write failures are tracked with exponential backoff. A resolver in
backoff is skipped while other resolvers are available, and a successful
response clears its backoff state.
