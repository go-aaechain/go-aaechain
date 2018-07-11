Pod::Spec.new do |spec|
  spec.name         = 'Gaae'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/aaechain/go-aaechain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS aaechain Client'
  spec.source       = { :git => 'https://github.com/aaechain/go-aaechain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gaae.framework'

	spec.prepare_command = <<-CMD
    curl https://gaaestore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gaae.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
