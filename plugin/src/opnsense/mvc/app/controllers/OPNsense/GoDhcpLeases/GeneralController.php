<?php

namespace OPNsense\GoDhcpLeases;

class GeneralController extends \OPNsense\Base\IndexController
{
    public function indexAction()
    {
        $this->view->generalForm = $this->getForm("general");
        $this->view->pick('OPNsense/GoDhcpLeases/general');
    }
}
