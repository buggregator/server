import {Meta, Story} from "@storybook/vue3";
import SmtpPageAddresses from '~/components/SmtpPageAddresses/SmtpPageAddresses.vue';

export default {
    title: "Smtp/Page/SmtpPageAddresses",
    component: SmtpPageAddresses
} as Meta<typeof SmtpPageAddresses>;

const Template: Story = (args) => ({
    components: {SmtpPageAddresses},
    setup() {
        return {
            args,
        };
    },
    template: `
      <SmtpPageAddresses v-bind="args"/>`,
});

export const EmailAddresses = Template.bind({});

EmailAddresses.args = {
    addresses: [
        {
            name: 'John Doe',
            email: 'john-doe@example.com',
        },
        {
            name: 'Jane Smith',
            email: 'JaneSmith@example.com',
        },
        {
            email: 'saraConor@example.com',
        }
    ],
};
