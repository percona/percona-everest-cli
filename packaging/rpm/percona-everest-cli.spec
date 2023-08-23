%global majorversion 0
%global minorversion 1
%global debug_package %{nil}
%global multiarch     %{nil}

# Define architecture-specific macros for multiarch builds
%ifarch x86_64
%global multiarch    x86_64
%endif
%ifarch aarch64
%global multiarch    aarch64
%endif

Summary:        CLI client for Percona Everest
Name:           percona-everest-cli
Version:        %{majorversion}.%{minorversion}
Release:        1%{?dist}
License:        ASL 2.0
Group:          Applications/Databases
URL:            https://github.com/percona/percona-everest-cli
Packager:       Percona Development Team <https://jira.percona.com>
Vendor:         Percona, LLC
Source0:        %{name}-%{version}.tar.gz

%description
This tool is a CLI client for Percona Everest and has the following features:
Provisioning of Percona Everest on Kubernetes clusters,
CLI client for Percona Everest that helps you manage database clusters
It is released under the Apache 2.0 license.

%prep
%setup -q
make init

%build
make release

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/%{_bindir}
%ifarch x86_64
cp dist/everestctl-linux-amd64 bin/everest
%endif
%ifarch aarch64
cp dist/everestctl-linux-arm64 bin/everest
%endif
%{__install} -p -D -m 0755 bin/everest %{buildroot}%{_bindir}/everest

%clean
rm -rf $RPM_BUILD_ROOT

%files
%{_bindir}/everest
%license LICENSE

%changelog
* Tue Aug 15 2023 Vadim Yalovets <vadim.yalovets> 0.1-1
- Initial build for percona-everest-cli
