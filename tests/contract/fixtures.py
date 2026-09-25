from client import GatewayClient


def create_group(
    gateway: GatewayClient,
    name: str,
    concept: str = "DATA",
) -> str:
    response = gateway.post(
        "/groups",
        json={"name": name, "concept": concept},
    )
    if response.status_code != 201:
        raise AssertionError(
            f"group creation returned {response.status_code}, want 201: {response.text}"
        )
    return response.json()["id"]
