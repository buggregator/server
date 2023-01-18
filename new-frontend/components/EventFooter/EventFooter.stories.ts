import EventFooter from "./EventFooter.vue";

export default {
  title: "Event/EventFooter",
  component: EventFooter,
};

const Template = (args: typeof Object) => ({
  components: { EventFooter },
  setup() {
    return {
      args,
    };
  },
  template: '<event-footer v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
  serverName: "My server",
  originConfig: {
    one: 1,
    two: 2,
  },
};
