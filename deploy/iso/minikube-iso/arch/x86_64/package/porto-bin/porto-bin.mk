################################################################################
#
# porto-bin
#
################################################################################

PORTO_BIN_VERSION = v5.3.33-alpha.3
PORTO_BIN_SITE = https://ytsaurus.hb.ru-msk.vkcs.cloud/porto
PORTO_BIN_SOURCE = porto-$(PORTO_BIN_VERSION).tgz

define PORTO_BIN_USERS
	- -1 porto -1 - - - - -
endef

define PORTO_BIN_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 \
		$(@D)/portod \
		$(TARGET_DIR)/sbin/portod
	$(INSTALL) -D -m 0755 \
		$(@D)/portoctl \
		$(TARGET_DIR)/sbin/portoctl
	$(INSTALL) -D -m 0755 \
		$(@D)/portoinit \
		$(TARGET_DIR)/sbin/portoinit
	$(INSTALL) -Dm644 \
		$(PORTO_BIN_PKGDIR)/k8s.conf \
		$(TARGET_DIR)/etc/portod.conf.d/k8s.conf
endef

define PORTO_BIN_INSTALL_INIT_SYSTEMD
	$(INSTALL) -D -m 644 \
		$(PORTO_BIN_PKGDIR)/porto.service \
		$(TARGET_DIR)/usr/lib/systemd/system/porto.service

	$(INSTALL) -D -m 644 \
		$(PORTO_BIN_PKGDIR)/porto.conf \
		$(TARGET_DIR)/etc/sysctl.d/porto.conf
endef

$(eval $(generic-package))
