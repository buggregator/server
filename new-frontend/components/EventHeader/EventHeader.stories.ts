import { action } from '@storybook/addon-actions'
import { EVENT_TYPES } from '~/config/constants';
import EventHeader from './EventHeader.vue';

export default {
  title: "Event/EventHeader",
  component: EventHeader,
  argTypes: {
    eventType: {
      control: { type: 'select' },
      options: Object.values(EVENT_TYPES),
      mapping: EVENT_TYPES
    },
  }
};

const Template = (args: typeof Object) => ({
  components: { EventHeader },
  methods: {
    action
  },
  setup() {
    return {
      args,
    };
  },
  template: `
    <event-header
      v-bind="args"
      @delete="(a) => action('Delete event')(a)"
      @toggle-view="(a) => action('Toggle event')(a)"
      @copy="(a) => action('Copied event')(a)"
      @download="(a) => action('Downloaded event')(a)"
    />
`,
});

export const Default = Template.bind({});
Default.args = {
  eventUrl: 'https://github.com/buggregator/spiral-app',
  eventType: EVENT_TYPES.SENTRY,
  isOpen: true,
  isVisibleControls: true,
  eventId: 'test-event-id',
  tags: ['one', 'two', 'tree']
};
