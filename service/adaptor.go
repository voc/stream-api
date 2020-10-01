type Stream interface {
	Subscribe() <- // get stream var updates
	Publish(plugin, interface{}) -> // publish var update

	GetCapacity()
}

type PluginAdaptor struct {

}

// Parse variables from template statements
func parseTemplate() {

}

func NewPluginAdaptor(claim *Claim, cfg, plugin, stream) *PluginAdaptor {
	pa := &PluginAdaptor{
		plugin: plugin.Create(stream)
		requiredVars: parseVars()
	}

	// handle.Update(plugin.vars)

	go func() {
		streamUpdate := stream.Subscribe()
		for {
			select {
			case <-ctx.Done():
				stream.Unsubscribe()
				return
			case state := <-streamUpdate:

			}
		}
	}

	pluginVars := template(cfg, stream)

	go func() {
		pluginUpdate := pa.plugin.Subscribe()
		for {
			select {
			case <-ctx.Done():
				pa.plugin.Unsubscribe()
				return
			case vars := <-pluginUpdate:
				extend(vars, pa.template(cfg.staticVars))

			}
		}
		func(vars) {

		})

	}()



	return pa
}

func (pa *PluginAdaptor) listen() {

}

func (pa *PluginAdaptor) template() {

}

func (pa *PluginAdaptor) createPlugin() {

}