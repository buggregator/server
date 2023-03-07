import { action } from '@storybook/addon-actions'
import { Meta, Story } from "@storybook/vue3";
import { EVENT_TYPES } from '~/config/constants';
import EventHeader from '~/components/EventHeader/EventHeader.vue';

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
} as Meta<typeof EventHeader>;

const Template: Story = (args) => ({
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
