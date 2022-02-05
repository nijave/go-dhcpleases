<?php

function godhcpleases_services()
{
	global $config;
	$services = array();

	if (isset($config['OPNsense']['GoDhcpLeases']['general']['enabled']) && $config['OPNsense']['GoDhcpLeases']['general']['enabled'] == 1) {
		$services[] = array(
			'description' => gettext('go-dhcpleases daemon'),
			'configd' => array(
				'restart' => array('go-dhcpleases restart'),
				'start' => array('go-dhcpleases start'),
				'stop' => array('go-dhcpleases stop'),
			),
			'name' => 'go-dhcpleases',
			'pidfile' => '/var/run/go-dhcpleases.pid'
		);
	}

	return $services;
}
