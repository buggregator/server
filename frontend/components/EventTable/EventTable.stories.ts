import { Meta, Story } from "@storybook/vue3";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default {
  title: "Components/EventTable",
  component: EventTable
} as Meta<typeof EventTable>;

const TableTemplate: Story = (args) => ({
  components: { EventTable, EventTableRow },
  setup() {
    return {
      args,
    };
  },
  template: `<EventTable>
    <EventTableRow title="Row 1">
      This is a row 1
    </EventTableRow>
    <EventTableRow title="Row 2">
      This is a row 2
    </EventTableRow>
    <EventTableRow title="Row 3">
      This is a row 3
    </EventTableRow>
  </EventTable>`,
});

export const Table = TableTemplate.bind({});
Table.args = {};
