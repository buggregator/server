import { normalizeMonologEvent } from "~/utils/normalizeEvent";
import EventMonolog from './EventMonolog.vue';
import monologEventMock from '../../mocks/monolog.json'

export default {
  title: "Event/EventMonolog",
  component: EventMonolog,
  argTypes: {
    eventType: {
      control: { type: 'object' },
    },
  }
};

const Template = (args: typeof Object) => ({
  components: { EventMonolog },
  setup() {
    return {
      args,
    };
  },
  template: `<event-monolog v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeMonologEvent(monologEventMock),
};
