import {Meta, Story} from "@storybook/vue3";
import EmailAttachmentsComponent from './EmailAttachments.vue';

export default {
    title: "Pages/Smtp/Parts",
    component: EmailAttachmentsComponent
} as Meta<typeof EmailAttachmentsComponent>;

const Template: Story = (args) => ({
    components: {EmailAttachmentsComponent},
    setup() {
        return {
            args,
        };
    },
    template: `
      <EmailAttachmentsComponent v-bind="args"/>`,
});

export const EmailAttachments = Template.bind({});

EmailAttachments.args = {
    attachments: [
        {
            name: 'Order.pdf',
        },
        {
            name: 'Invoice.pdf',
        },
        {
            name: 'Commercial offer.pdf',
        }
    ],
};