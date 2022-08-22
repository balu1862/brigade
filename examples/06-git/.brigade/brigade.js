const { events, Job, Container } = require("@brigadecore/brigadier");

events.on("brigade.sh/cli", "exec", async event => {
  let job = new Job("dind", "docker:stable-dind", event);
  job.primaryContainer.environment.DOCKER_HOST = "localhost:2375";
  job.primaryContainer.command = ["sh"];
  job.primaryContainer.arguments = [
    "-c",
    // Wait for the Docker daemon to start up
    // And then pull the image
    //"sleep 20 && docker pull busybox"
    "sleep 1"
  ];

  // Run the Docker daemon in a sidecar container
  job.sidecarContainers = {
    "docker": new Container("docker:stable-dind")
  };
  job.sidecarContainers.docker.privileged = true
  job.sidecarContainers.command = ["sh"];
  job.sidecarContainers.arguments = [
    "-c",
    // Wait for the Docker daemon to start up
    // And then pull the image
    "sleep 20 && docker pull busybox"
  ];

  await job.run();
});

events.process();
