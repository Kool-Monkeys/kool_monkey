Summary: Kool monkey is a distributed system to monitor webpages. This package provides the server part of Kool monkey.
Name: kool-server
Version: %{version}
Release: %{release}
License: GPLv3
Group: Applications/Multimedia
Requires: postgresql94
URL: https://github.com/gophergala2016/kool_monkey

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root
# Should work in centos, but it's not working in debian :(
#BuildRequires: golang
Requires(post): /sbin/chkconfig /usr/sbin/useradd
Requires(preun): /sbin/chkconfig, /sbin/service
Requires(postun): /sbin/service
Provides: kool-server

%description
Kool monkey is a distributed system to monitor webpages. This package
provides the server part of Kool monkey.

%build
PROD_BUILD=1 %{__make} kool-server
PROD_BUILD=1 %{__make} install

%install
%{__rm} -rf %{buildroot}
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_sysconfdir}
mkdir -p %{buildroot}%{_datadir}
mkdir -p %{buildroot}%{_exec_prefix}/www
mkdir -p %{buildroot}%{_exec_prefix}/dashboard
%{__install} -Dp -m 0755 bin/kool-server %{buildroot}%{_bindir}/kool-server
%{__install} -Dp -m 0644 dev-env/scripts/create_db.sql %{buildroot}%{_datadir}
%{__install} -Dp -m 0644 dev-env/scripts/upgrade_db.sql %{buildroot}%{_datadir}
%{__install} -Dp -m 0644 dev-env/scripts/01_create_roles.sql %{buildroot}%{_datadir}
%{__install} -Dp -m 0644 dev-env/scripts/02_create_schema.sql %{buildroot}%{_datadir}
%{__install} -Dp -m 0644 front/www/* %{buildroot}%{_exec_prefix}/www
%{__install} -Dp -m 0644 front/dashboard/* %{buildroot}%{_exec_prefix}/dashboard
%{__install} -Dp -m 0755 scripts/init/kool-server %{buildroot}%{_sysconfdir}/init.d/kool-server
%{__install} -Dp -m 0755 systemd/kool-server.service %{buildroot}%{_systemddir}/kool-server.service
%{__install} -Dp -m 0644 dev-env/conf/kool-server.conf %{buildroot}%{_sysconfdir}/kool-server.conf
%{__install} -p -d -m 0755 %{buildroot}%{pid_dir}

%pre
/usr/sbin/useradd -c 'monkey' -u 499 -s /bin/false -r -d %{_prefix} monkey 2> /dev/null || :

%preun
if [ $1 = 0 ]; then
    term="/dev/$(ps -p$$ --no-heading | awk '{print $2}')"
    exec < $term

    /bin/systemctl stop kool-server.service
    /bin/systemctl disable kool-server.service
else
    term="/dev/$(ps -p$$ --no-heading | awk '{print $2}')"
    exec < $term

    /bin/systemctl stop kool-server.service
fi

%post
port="-p 5430"
if [ -z `su postgres -c "/usr/bin/psql ${port} -l | grep monkey"` ]; then
	su postgres -c "/usr/bin/createdb ${port} monkey"
	su postgres -c "/usr/bin/psql ${port} monkey -f %{_datadir}/create_db.sql"
else
	echo "There's a previous monkey database, trying to upgrade it"
	su postgres -c "/usr/bin/psql ${port} monkey -f %{_datadir}/upgrade_db.sql"
fi
/bin/systemctl daemon-reload
/bin/systemctl enable kool-server.service
/bin/systemctl start kool-server

%postun
port="-p 5430"
if [ $1 = 0 ]; then
	su postgres -c "/usr/bin/dropdb ${port} --if-exists monkey"
else
	su postgres -c "/usr/bin/psql ${port} monkey -f %{_datadir}/upgrade_db.sql"
fi
	
%clean
%{__rm} -rf %{buildroot}

%files
%defattr(-, root, root, 0755)
%{_bindir}/kool-server
%{_exec_prefix}/www
%{_exec_prefix}/dashboard
%{_datadir}/create_db.sql
%{_datadir}/upgrade_db.sql
%{_datadir}/01_create_roles.sql
%{_datadir}/02_create_schema.sql
%{_sysconfdir}/init.d/kool-server
%{_systemddir}/kool-server.service
%config(noreplace) %{_sysconfdir}/kool-server.conf

%changelog
* Sun Jan 24 2016 Pablo Alvarez de Sotomayor Posadillo <palvarez@ritho.net> 0.2-0
- Fix the sql paths to create the database during the installation.
- Add a build option to generate the configuration for production.

* Sat Jan 23 2016 Pablo Alvarez de Sotomayor Posadillo <palvarez@ritho.net> 0.1-0
- First version of the rpm package.
