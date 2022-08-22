const { events, Job, Container } = require("@brigadecore/brigadier");

events.on("brigade.sh/cli", "exec", async event => {
  let job = new Job("dind", "docker:stable-dind", event);
  let keys = Object.keys(job.primaryContainer)
  console.log(keys);
  job.primaryContainer.privileged = true;
  job.primaryContainer.environment.DOCKER_HOST = "tcp://0.0.0.0:2376";
  job.primaryContainer.command = ["sh"];
  job.primaryContainer.arguments = [
    "-c",
    // Wait for the Docker daemon to start up
    // And then pull the image
    "sleep 30 && docker pull busybox"
  ];

  // Run the Docker daemon in a sidecar container
  job.sidecarContainers = {
    "docker": new Container("docker:stable-dind")
  };
  job.sidecarContainers.docker.privileged = true
  await job.run();
});

events.process();
