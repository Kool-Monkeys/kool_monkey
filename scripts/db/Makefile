TOPDIR?=$(realpath ../..)
include $(TOPDIR)/Config.mk

DIRS  =
SRCS  = create_db.sql
SRCS += upgrade_db.sql
SRCS += 00_create_roles.sql
SRCS += 01_create_schema.sql
SRCS += 99_insert_test_data.sql

install:
	@echo "\\033[1;35m+++ Installing conf\\033[39;0m"
	@mkdir -p $(SCRIPTS_DIR)
	@for i in $(DIRS) ; do $(MAKE) -C $$i install ; done
	@for i in $(SRCS); do \
		$(CP) $$i $(SCRIPTS_DIR) ; \
		perl -i -pne 's/%(\w+?)%/$$ENV{$$1}/ge' $(SCRIPTS_DIR)/$$i ; \
	done
	@echo "\\033[1;35m+++ Conf installed\\033[39;0m"
