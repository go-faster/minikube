################################################################################
#
# portoshim-bin
#
################################################################################

PORTOSHIM_BIN_VERSION = v1.0.11-alpha.7
PORTOSHIM_BIN_SITE = https://github.com/go-faster/portoshim/releases/download/$(PORTOSHIM_BIN_VERSION)
PORTOSHIM_BIN_SOURCE = portoshim_focal_$(PORTOSHIM_BIN_VERSION)_amd64.tgz

define PORTOSHIM_BIN_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 \
		$(@D)/portoshim \
		$(TARGET_DIR)/sbin/portoshim
	$(INSTALL) -D -m 0755 \
		$(@D)/logshim \
		$(TARGET_DIR)/sbin/logshim
	$(INSTALL) -D -m 644 \
		$(PORTOSHIM_BIN_PKGDIR)/crictl.yaml \
		$(TARGET_DIR)/etc/crictl.yaml
endef

define PORTOSHIM_BIN_INSTALL_INIT_SYSTEMD
	$(INSTALL) -D -m 644 \
		$(PORTOSHIM_BIN_PKGDIR)/portoshim.service \
		$(TARGET_DIR)/usr/lib/systemd/system/portoshim.service
endef

$(eval $(generic-package))
