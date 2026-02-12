Vagrant.configure("2") do |config|
  config.vm.box = "generic/alpine321"
  config.vm.hostname = "scoututm-dev"

  config.vm.provider "virtualbox" do |vb|
    vb.cpus = 4
    vb.memory = 8048
    vb.name = "scoututm-dev"
  end

  # Port forwards — map host ports to VM dev-mode ports.
  config.vm.network "forwarded_port", guest: 443,  host: 8443, host_ip: "127.0.0.1"
  config.vm.network "forwarded_port", guest: 80,   host: 8080, host_ip: "127.0.0.1"

  # Private network for simulated LAN.
  config.vm.network "private_network", ip: "192.168.56.10"

  # Sync build artifacts into the VM.
  config.vm.synced_folder "build/", "/opt/scout-build", type: "rsync",
    rsync__exclude: [".git/"]

  # Provision with ScoutOS packages.
  config.vm.provision "shell", path: "tools/vagrant-provision.sh"
end
