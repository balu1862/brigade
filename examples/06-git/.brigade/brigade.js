const { events, Job } = require("@brigadecore/brigadier");

events.on("brigade.sh/cli", "exec", async event => {
  let job = new Job("dood", "docker", event);
  let keys = Object.keys(job.primaryContainer)
  console.log(keys);
  job.primaryContainer.useHostDockerSocket = true;
  job.primaryContainer.privileged = true;
  job.primaryContainer.command = ["docker"];
  job.primaryContainer.arguments = ["ps"];
  await job.run();
});

events.process();
