# AWS setup checklist

The minimum cloud deployment for this demo is **AWS IoT Core only**. Go, PostgreSQL, and Flutter can stay on your laptop. No EC2, RDS, Lambda, MSK, Redis, or public inbound port is needed for this first demonstration. AWS resources have not been provisioned by this project.

## 1. Choose a region and a budget

- [ ] Sign in with your normal AWS administrator/developer identity.
- [ ] Select one region and keep all IoT resources there. For example, Mumbai is `ap-south-1`; this is a choice, not a requirement.
- [ ] Record your region and 12-digit AWS account ID for the policy templates.
- [ ] Set a small billing budget/notification suitable for your account, and stop the simulator after the demo. Budgets notify rather than acting as a universal spending cap. Do not assume the demo is free.

## 2. Create separate IoT policies

In AWS IoT Core, find **Security → Policies** (some console versions group this under **Secure**).

- [ ] Open `infra/aws/simulator-policy.json`. Replace **both** `REGION` and `ACCOUNT_ID` everywhere. Create an IoT policy named `MoonriderSimulatorPolicy` with that document.
- [ ] Do the same with `infra/aws/backend-policy.json`; create `MoonriderBackendPolicy`.

These are **IoT policies attached to device certificates**, not IAM role trust policies. The simulator can only connect as `moonrider-simulator` and publish to one telemetry topic. The backend can only connect as `moonrider-backend`, subscribe to that topic, and receive its data. It cannot publish. `Subscribe` uses a `topicfilter` ARN; `Publish` and `Receive` use `topic` ARNs. No resource wildcard is necessary.

## 3. Register the simulator and download its credentials

In IoT Core, find **Manage → All devices → Things** (labels may vary), then **Create things → Create single thing**.

- [ ] Name the thing `moonrider-simulator`.
- [ ] Choose to generate a new X.509 certificate.
- [ ] Download its device certificate and private key. The generated private key must be saved when offered.
- [ ] Ensure the certificate is **Active**.
- [ ] Attach `MoonriderSimulatorPolicy` to this certificate.
- [ ] Ensure the certificate is associated with the simulator thing.
- [ ] Save the files locally as `certs/simulator.pem` and `certs/simulator.key`.

The payload's tractor ID is `MR-001`; the AWS thing name represents the simulator's connection identity. They need not be identical.

## 4. Register the backend separately

- [ ] Create another thing named `moonrider-backend` and generate a **different** active certificate/key pair.
- [ ] Attach **only** `MoonriderBackendPolicy` to its certificate.
- [ ] Associate it with the backend thing.
- [ ] Save the files as `certs/backend.pem` and `certs/backend.key`.

Do not reuse the simulator private key for the backend. MQTT client IDs must also be distinct; two simultaneous clients with the same ID can disconnect one another.

## 5. Download the root CA and copy the endpoint

- [ ] Download [Amazon Root CA 1](https://www.amazontrust.com/repository/AmazonRootCA1.pem) to `certs/AmazonRootCA1.pem`.
- [ ] In the IoT Core **Settings / Device data endpoint** area, copy your account's **ATS data endpoint**, typically `xxxxxxxx-ats.iot.REGION.amazonaws.com`.

Alternatively, with an already authenticated AWS CLI:

```sh
aws iot describe-endpoint --endpoint-type iot:Data-ATS --region YOUR_REGION
```

Use `ssl://YOUR-ATS-ENDPOINT:8883` for `MQTT_BROKER`. Do not prepend `https://`. No AWS access key is needed inside the Flutter app or either MQTT client: the data connection uses the client certificate and private key.

## 6. Run PostgreSQL locally

From the project root:

```sh
docker compose up -d postgres
```

Stop any API or simulator already running. The AWS mode below uses Go on your host, not the Compose API service. Keep port 8080 available.

## 7. Configure and run the backend

From the project root, substitute your own endpoint and absolute certificate paths:

```sh
export DATABASE_URL='postgres://moonrider:local-demo-only@127.0.0.1:5432/moonrider?sslmode=disable'
export MQTT_BROKER='ssl://YOUR-ATS-ENDPOINT:8883'
export MQTT_CLIENT_ID='moonrider-backend'
export MQTT_CERT='/absolute/path/to/moonrider/certs/backend.pem'
export MQTT_KEY='/absolute/path/to/moonrider/certs/backend.key'
export MQTT_CA='/absolute/path/to/moonrider/certs/AmazonRootCA1.pem'
cd backend
go run ./cmd/api
```

- [ ] Confirm the log says `MQTT subscribed`.
- [ ] Open `http://127.0.0.1:8080/readyz`; it should report `ready` once the subscription and PostgreSQL are available.

## 8. Run the simulator in a NEW terminal

Environment variables are terminal-specific. Set the broker/CA again, but use the simulator certificate and ID:

```sh
export MQTT_BROKER='ssl://YOUR-ATS-ENDPOINT:8883'
export MQTT_CLIENT_ID='moonrider-simulator'
export MQTT_CERT='/absolute/path/to/moonrider/certs/simulator.pem'
export MQTT_KEY='/absolute/path/to/moonrider/certs/simulator.key'
export MQTT_CA='/absolute/path/to/moonrider/certs/AmazonRootCA1.pem'
cd backend
go run ./cmd/simulator --scenario cycle --duplicate
```

## 9. Verify the complete demonstration

- [ ] In IoT Core's **MQTT test client**, subscribe to `moonrider/tractors/MR-001/telemetry`. Observe JSON readings. This console session uses your AWS console identity; do not broaden the device certificate policies to make it work.
- [ ] From the project root, start `flutter run -d chrome --web-port 5173`.
- [ ] Confirm battery, temperature, and speed update.
- [ ] Observe one overheating episode, its recovery, low battery/recovery, and one connectivity-loss/recovery episode.
- [ ] Confirm `--duplicate` does not multiply alerts.
- [ ] Capture the app and console together for your interview recording, hiding account details and never showing private keys.

If connection fails, check the active certificate, attached IoT policy, substituted region/account, ATS endpoint, exact client ID, root CA, and outbound TCP port 8883. If publish works but the backend receives nothing, check its **Subscribe and Receive** permissions and its subscription readiness log.

## 10. Finish and clean up

- [ ] Stop both Go processes to stop publishing/receiving.
- [ ] Stop local containers with `docker compose stop` if no longer needed. This preserves local data.
- [ ] For a disposable demo, deactivate the two device certificates to disable connections. Remove the things, policies, and certificates through the console when you no longer need them; detach associations first as required by AWS.
- [ ] Keep keys out of Git. `certs/`, `.env`, PEMs, and key files are ignored. Do not paste private keys into chat or include them in screenshots.

## Optional later cloud hosting

Once IoT Core works, the next useful extension is one Go instance on EC2 and PostgreSQL on RDS. Before public access, add HTTPS/WSS, application authentication/authorization, managed secrets, a private database security group, backups, and monitoring. The shipped API intentionally defaults to loopback and is not suitable for public exposure without these controls. This is a future deployment design, not part of the required AWS checklist.

## Official references

- [Create an IoT thing, certificate, and private key](https://docs.aws.amazon.com/iot/latest/developerguide/iot-moisture-create-thing.html)
- [AWS IoT connection and topic policy examples](https://docs.aws.amazon.com/iot/latest/developerguide/connection-policy.html)
- [Connect devices and find endpoints](https://docs.aws.amazon.com/iot/latest/developerguide/iot-connect-devices.html)
- [MQTT QoS and persistent sessions](https://docs.aws.amazon.com/iot/latest/developerguide/mqtt.html)
