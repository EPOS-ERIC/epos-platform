import unittest
from unittest.mock import Mock, patch

import requests

from client import GatewayClient


class GatewayClientTest(unittest.TestCase):
    def test_request_encodes_path_and_validates_response(self) -> None:
        """Verify path encoding and response validation."""
        operation = Mock()
        response = Mock(spec=requests.Response)
        schema = {"/resources/{resourceType}/{id}": {"GET": operation}}
        session = requests.Session()

        client = GatewayClient(
            schema,
            "http://gateway/v1/",
            "token",
            session=session,
        )
        with patch.object(session, "request", return_value=response) as request:
            result = client.get(
                "/resources/{resourceType}/{id}",
                resourceType="software-application",
                id="resource/id",
            )

        self.assertIs(result, response)
        request.assert_called_once_with(
            "GET",
            "http://gateway/v1/resources/software-application/resource%2Fid",
            params=None,
            json=None,
            data=None,
            headers=None,
            timeout=30,
        )
        operation.validate_response.assert_called_once_with(response)

    def test_request_rejects_missing_path_parameter(self) -> None:
        """Verify missing path parameters fail before the HTTP request."""
        operation = Mock()
        schema = {"/resources/{id}": {"GET": operation}}
        client = GatewayClient(schema, "http://gateway/v1", "token")

        with self.assertRaisesRegex(ValueError, "missing path parameter"):
            client.get("/resources/{id}")
