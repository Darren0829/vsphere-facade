package virtualmachinecustomize

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"net"
	"vsphere-facade/app/logging"
	"vsphere-facade/helper"
	"vsphere-facade/helper/virtualmachine"
)

type CustomizeParameter struct {
	OSSetting       *OSSettingParameter `json:"OSSetting"`
	GlobalIPSetting *GlobalIPSetting    `json:"globalIPSetting"`
	NicSetting      []*NicSetting       `json:"nicSetting"`
}

type OSSettingParameter struct {
	LinuxSetting   *LinuxSettingParameter   `json:"linuxSetting"`
	WindowsSetting *WindowsSettingParameter `json:"windowsSetting"`
}

type LinuxSettingParameter struct {
	HostName   *string `json:"hostName,omitempty"`
	Domain     *string `json:"domain,omitempty"`
	TimeZone   *string `json:"timeZone,omitempty"`
	HwClockUTC *bool   `json:"hwClockUTC,omitempty"`
	ScriptText *string `json:"scriptText,omitempty"`
}

type WindowsSettingParameter struct {
	ComputerName   string   `json:"computerName,omitempty"`
	Password       string   `json:"password,omitempty"`
	TimeZone       int32    `json:"timeZone,omitempty"`
	AutoLogon      bool     `json:"autoLogon,omitempty"`
	AutoLogonCount int32    `json:"autoLogonCount,omitempty"`
	Commands       []string `json:"commands,omitempty"`

	Register struct {
		FullName string `json:"fullName"`
		OrgName  string `json:"orgName"`
	} `json:"register,omitempty"`

	JoinWorkGroup *string `json:"joinWorkGroup,omitempty"`
	JoinDomain    *struct {
		Domain              string `json:"domain"`
		DomainAdmin         string `json:"domainAdmin"`
		DomainAdminPassword string `json:"domainAdminPassword"`
	} `json:"joinDomain,omitempty"`

	ProductID string `json:"productId,omitempty"`
}

type GlobalIPSetting struct {
	Gateway       *string   `json:"gateway,omitempty"`
	DnsSuffixList *[]string `json:"dnsSuffixList,omitempty"`
	DnsServerList *[]string `json:"dnsServerList,omitempty"`
}

type NicSetting struct {
	Key           int32
	DnsServerList []string
	DnsDomain     *string
	Gateway       []string
	SubnetMask    *int32
	IPv4          *NicIPv4Setting
	IPv6          *NicIPv6Setting
}

type NicIPv4Setting struct {
	Static    bool    `json:"static"`
	IPAddress *string `json:"ipAddress,omitempty"`
}

type NicIPv6Setting struct {
	Gateways []string `json:"gateways,omitempty"`
	IPv6s    *[]IPv6  `json:"IPv6s"`
}

type IPv6 struct {
	Static     bool
	IPAddress  *string
	SubnetMask *int32
}

func Customize(api *helper.API, ID string, p *CustomizeParameter) error {
	logging.L().Debug(fmt.Sprintf("修改虚拟机[%s]系统配置", ID))
	oVM := virtualmachine.GetObject(api, ID)
	if oVM == nil {
		return fmt.Errorf("虚拟机[%s]系统配置失败，虚拟机不存在", ID)
	}

	var spec = types.CustomizationSpec{}
	var err error
	err = osSetting(api, oVM, p, &spec)
	if err != nil {
		return fmt.Errorf("设置虚拟机[%s]OS信息失败: %s", ID, err)
	}

	err = globalIPSettings(p, &spec)
	if err != nil {
		return fmt.Errorf("设置虚拟机[%s]全局IP信息失败: %s", ID, err)
	}

	err = nicSetting(oVM, p, &spec)
	if err != nil {
		return fmt.Errorf("设置虚拟机[%s]网卡信息失败: %s", ID, err)
	}

	err = waitCustomizeTask(oVM, spec)
	if err != nil {
		return fmt.Errorf("虚拟机[%s]系统设置任务失败: %s", ID, err)
	}
	logging.L().Debug(fmt.Sprintf("修改虚拟机[%s]系统配置完成", ID))
	return nil
}

func waitCustomizeTask(oVM *object.VirtualMachine, spec types.CustomizationSpec) error {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	oTask, err := oVM.Customize(ctx, spec)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer tcancel()
	return oTask.Wait(tctx)
}

// osSetting
// 对应vim.vm.customization.Specification: identity
func osSetting(api *helper.API, oVM *object.VirtualMachine, p *CustomizeParameter, spec *types.CustomizationSpec) error {
	if p.OSSetting == nil {
		return nil
	}

	OSInfo := virtualmachine.GetOSInfo(api, oVM)
	switch OSInfo.GuestFamily {
	case string(types.VirtualMachineGuestOsFamilyLinuxGuest):
		err := linuxOSSetting(p.OSSetting.LinuxSetting, spec)
		if err != nil {
			return fmt.Errorf("虚拟机[%s]操作系统配置失败: %s", oVM.Reference().Value, err.Error())
		}
	case string(types.VirtualMachineGuestOsFamilyWindowsGuest):
		err := windowsOSSetting(p.OSSetting.WindowsSetting, spec)
		if err != nil {
			return fmt.Errorf("虚拟机[%s]操作系统配置失败: %s", oVM.Reference().Value, err.Error())
		}
	}
	return nil
}

// globalIPSettings
// 对应vim.vm.customization.Specification: globalIPSettings
func globalIPSettings(p *CustomizeParameter, spec *types.CustomizationSpec) error {
	if p.GlobalIPSetting == nil {
		return nil
	}
	globalIPSetting := *p.GlobalIPSetting
	ipSetting := types.CustomizationGlobalIPSettings{}
	if globalIPSetting.DnsSuffixList != nil {
		ipSetting.DnsSuffixList = *globalIPSetting.DnsSuffixList
	}
	if globalIPSetting.DnsServerList != nil {
		ipSetting.DnsServerList = *globalIPSetting.DnsServerList
	}
	spec.GlobalIPSettings = ipSetting
	return nil
}

// nicSetting
// 对应vim.vm.customization.Specification: nicSettingMap
func nicSetting(oVM *object.VirtualMachine, p *CustomizeParameter, spec *types.CustomizationSpec) error {
	if p.NicSetting == nil {
		return nil
	}
	var adapterMappings []types.CustomizationAdapterMapping
	for _, n := range p.NicSetting {
		var adapterMapping = types.CustomizationAdapterMapping{}
		var IPSettings = types.CustomizationIPSettings{}

		if n.DnsDomain != nil {
			IPSettings.DnsDomain = *n.DnsDomain
		}
		if n.SubnetMask != nil {
			IPSettings.SubnetMask = v4CIDRMaskToDotted(int(*n.SubnetMask))
		}
		if n.DnsServerList != nil {
			IPSettings.DnsServerList = n.DnsServerList
		}
		if n.IPv4 != nil {
			IPSettings.Ip = ipv4Setting(n.IPv4)
		}
		if n.IPv6 != nil {
			IPSettings.IpV6Spec = ipv6Setting(n.IPv6)
		}
		if n.Gateway != nil {
			IPSettings.Gateway = n.Gateway
		} else {
			if p.GlobalIPSetting != nil && p.GlobalIPSetting.Gateway != nil {
				globalGateway := *p.GlobalIPSetting.Gateway
				gatewayIp := net.ParseIP(globalGateway)
				if gatewayIp.To4() != nil && n.IPv4 != nil && n.IPv4.IPAddress != nil {
					ip := *n.IPv4.IPAddress
					matched := matchGateway(ip, int(*n.SubnetMask), globalGateway)
					if matched {
						IPSettings.Gateway = []string{globalGateway}
					}
				}
			}
		}
		adapterMapping.Adapter = IPSettings
		adapterMappings = append(adapterMappings, adapterMapping)
	}
	if len(adapterMappings) > 0 {
		spec.NicSettingMap = adapterMappings
	}
	return nil
}

func v4CIDRMaskToDotted(mask int) string {
	m := net.CIDRMask(mask, 32)
	a := int(m[0])
	b := int(m[1])
	c := int(m[2])
	d := int(m[3])
	return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
}

func matchGateway(a string, m int, g string) bool {
	ip := net.ParseIP(a)
	gw := net.ParseIP(g)
	var mask net.IPMask
	if ip.To4() != nil {
		mask = net.CIDRMask(m, 32)
	} else {
		mask = net.CIDRMask(m, 128)
	}
	if ip.Mask(mask).Equal(gw.Mask(mask)) {
		return true
	}
	return false
}

func ipv4Setting(p *NicIPv4Setting) types.BaseCustomizationIpGenerator {
	if p == nil {
		return nil
	}

	if p.Static && p.IPAddress != nil {
		logging.L().Debug(fmt.Sprintf("设置固定IP：[%s]", *p.IPAddress))
		return &types.CustomizationFixedIp{
			IpAddress: *p.IPAddress,
		}
	} else {
		return &types.CustomizationDhcpIpGenerator{}
	}
}

func ipv6Setting(p *NicIPv6Setting) *types.CustomizationIPSettingsIpV6AddressSpec {
	if p == nil {
		return nil
	}
	var spec types.CustomizationIPSettingsIpV6AddressSpec
	spec.Gateway = p.Gateways

	var ipV6Generator []types.BaseCustomizationIpV6Generator
	ips := *p.IPv6s
	for _, ip := range ips {
		if ip.Static {
			logging.L().Debug(fmt.Sprintf("设置固定IP：[%s]", *ip.IPAddress))
			fixedIpV6 := types.CustomizationFixedIpV6{
				IpAddress:  *ip.IPAddress,
				SubnetMask: *ip.SubnetMask,
			}
			ipV6Generator = append(ipV6Generator, &fixedIpV6)
		} else {
			ipV6Generator = append(ipV6Generator, &types.CustomizationDhcpIpV6Generator{})
		}
	}
	return &spec
}

func linuxOSSetting(p *LinuxSettingParameter, spec *types.CustomizationSpec) error {
	if p == nil {
		return nil
	}

	var linux = types.CustomizationLinuxPrep{}
	if p.HostName != nil {
		linux.HostName = &types.CustomizationFixedName{
			Name: *p.HostName,
		}
	}
	if p.Domain != nil {
		linux.Domain = *p.Domain
	}
	if p.HwClockUTC != nil {
		linux.HwClockUTC = p.HwClockUTC
	}
	if p.TimeZone != nil {
		linux.TimeZone = *p.TimeZone
	}
	if p.ScriptText != nil {
		linux.ScriptText = *p.ScriptText
	}
	spec.Identity = &linux
	return nil
}

func windowsOSSetting(p *WindowsSettingParameter, spec *types.CustomizationSpec) error {
	if p == nil {
		return nil
	}
	var windows = types.CustomizationSysprep{}

	// 用户密码
	guiUnattended := types.CustomizationGuiUnattended{
		TimeZone:       p.TimeZone,
		AutoLogon:      p.AutoLogon,
		AutoLogonCount: p.AutoLogonCount,
	}
	if p.Password != "" {
		guiUnattended.Password = &types.CustomizationPassword{
			Value:     p.Password,
			PlainText: true,
		}
	}
	windows.GuiUnattended = guiUnattended

	// 初次登录执行命令
	if len(p.Commands) > 0 {
		windows.GuiRunOnce = &types.CustomizationGuiRunOnce{
			CommandList: p.Commands,
		}
	}

	// 加入域或者工作组
	var identification = types.CustomizationIdentification{}
	if p.JoinWorkGroup != nil {
		identification.JoinWorkgroup = *p.JoinWorkGroup
	} else if p.JoinDomain != nil {
		identification.JoinDomain = p.JoinDomain.Domain
		identification.DomainAdmin = p.JoinDomain.DomainAdminPassword
		identification.DomainAdminPassword = &types.CustomizationPassword{
			Value:     p.JoinDomain.DomainAdminPassword,
			PlainText: true,
		}
	}
	windows.Identification = identification

	userData := types.CustomizationUserData{}
	userData.OrgName = p.Register.OrgName
	userData.FullName = p.Register.FullName

	if p.ComputerName != "" {
		userData.ComputerName = &types.CustomizationFixedName{
			Name: p.ComputerName,
		}
	}
	if p.ProductID != "" {
		userData.ProductId = p.ProductID
	}
	windows.UserData = userData

	spec.Identity = &windows
	return nil
}
