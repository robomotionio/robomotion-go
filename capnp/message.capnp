using Go = import "/go.capnp";
@0x85d3acc39d94e0f8;
$Go.package("robocapnp");
$Go.import("robocapnp");

struct NodeMessage {
    content @0 :Data;
    # The whole message.
}
