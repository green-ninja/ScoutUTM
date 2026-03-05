Vagrant.configure("2") do |config|
  # ── UTM Appliance VM (primary) ──
  config.vm.define "utm", primary: true do |utm|
    utm.vm.box = "generic/alpine319"
    utm.vm.hostname = "scoututm-dev"

    utm.vm.provider "virtualbox" do |vb|
      vb.cpus = 4
      vb.memory = 8048
      vb.name = "scoututm-dev"
    end

    # Port forwards — map host ports to VM ports.
    utm.vm.network "forwarded_port", guest: 443,  host: 8443, host_ip: "127.0.0.1"
    utm.vm.network "forwarded_port", guest: 80,   host: 8080, host_ip: "127.0.0.1"

    # LAN interface on internal network (matches default config 192.168.1.0/24).
    utm.vm.network "private_network", ip: "192.168.1.1", netmask: "255.255.255.0", virtualbox__intnet: "scoutlan"

    # Sync build artifacts and project root into the VM.
    utm.vm.synced_folder "build/", "/opt/scout-build"
    utm.vm.synced_folder "tests/", "/opt/scout-tests"

    # Provision with ScoutOS packages + optionally run tests.
    utm.vm.provision "shell", path: "tools/vagrant-provision.sh",
      env: { "SCOUT_NO_TESTS" => ENV.fetch("SCOUT_NO_TESTS", "") }
  end

  # ── LAN Client VM ──
  config.vm.define "client" do |client|
    client.vm.box = "generic/alpine319"
    client.vm.hostname = "scout-client"

    client.vm.provider "virtualbox" do |vb|
      vb.cpus = 1
      vb.memory = 512
      vb.name = "scout-client"
    end

    # LAN interface — DHCP from ScoutUTM's dnsmasq (no auto_config so Vagrant
    # doesn't assign a static IP; the provisioning script gets one via DHCP).
    client.vm.network "private_network", auto_config: false, virtualbox__intnet: "scoutlan"

    # Provision: DHCP, routing, connectivity checks.
    client.vm.provision "shell", path: "tools/client-provision.sh"
  end
end
