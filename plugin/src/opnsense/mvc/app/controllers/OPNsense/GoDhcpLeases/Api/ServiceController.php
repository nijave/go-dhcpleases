<?php

namespace OPNsense\GoDhcpLeases\Api;

use OPNsense\Base\ApiMutableServiceControllerBase;
use OPNsense\Core\Backend;
use OPNsense\GoDhcpLeases\General;

/**
 * Class ServiceController
 * @package OPNsense\GoDhcpLeases
 */
class ServiceController extends ApiMutableServiceControllerBase
{
    protected static $internalServiceClass = '\OPNsense\GoDhcpLeases\General';
    protected static $internalServiceTemplate = 'OPNsense/GoDhcpLeases';
    protected static $internalServiceEnabled = 'enabled';
    protected static $internalServiceName = 'go-dhcpleases';
}
