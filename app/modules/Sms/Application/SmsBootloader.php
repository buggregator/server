<?php

declare(strict_types=1);

namespace Modules\Sms\Application;

use App\Application\Event\EventTypeRegistryInterface;
use Modules\Sms\Application\Gateway\FieldMapGateway;
use Modules\Sms\Application\Gateway\GatewayRegistry;
use Modules\Sms\Application\Mapper\EventTypeMapper;
use Spiral\Boot\Bootloader\Bootloader;

final class SmsBootloader extends Bootloader
{
    public function defineSingletons(): array
    {
        return [
            GatewayRegistry::class => static fn(): GatewayRegistry => new GatewayRegistry([
                // ===== Providers with highly unique detect fields (order doesn't matter) =====

                // Twilio: MessageSid is unique
                new FieldMapGateway('twilio', ['MessageSid', 'Body'], ['From'], ['To'], ['Body']),
                // Plivo: MessageUUID is unique
                new FieldMapGateway('plivo', ['MessageUUID'], ['From', 'src'], ['To', 'dst'], ['Text', 'text']),
                // Sinch: batch_id is unique
                new FieldMapGateway('sinch', ['batch_id', 'body'], ['from'], ['to'], ['body']),
                // Vonage: api_key+api_secret pair
                new FieldMapGateway('vonage', ['api_key', 'api_secret'], ['from', 'msisdn'], ['to'], ['text']),
                // Infobip: nested messages array
                new FieldMapGateway('infobip', ['messages'], ['from'], ['to'], ['text']),
                // MessageBird: originator is unique field name
                new FieldMapGateway('messagebird', ['originator', 'recipients'], ['originator'], ['recipients'], ['body']),
                // Telnyx: messaging_profile_id is unique
                new FieldMapGateway('telnyx', ['messaging_profile_id'], ['from'], ['to'], ['text']),
                // Bandwidth: applicationId is unique
                new FieldMapGateway('bandwidth', ['applicationId', 'text'], ['from'], ['to'], ['text']),
                // Brevo: sender+recipient+content triple
                new FieldMapGateway('brevo', ['sender', 'recipient', 'content'], ['sender'], ['recipient'], ['content']),
                // Termii: api_key+sms combo
                new FieldMapGateway('termii', ['api_key', 'sms'], ['from'], ['to'], ['sms']),
                // SmsFactor: gsmsmsid is unique
                new FieldMapGateway('smsfactor', ['gsmsmsid'], ['sender'], ['to'], ['text']),
                // MessageMedia: source_number+destination_number pair
                new FieldMapGateway('messagemedia', ['source_number', 'destination_number'], ['source_number'], ['destination_number'], ['content']),
                // Lox24: sender_id+phone pair
                new FieldMapGateway('lox24', ['sender_id', 'phone'], ['sender_id'], ['phone'], ['text']),
                // Unifonic: AppSid is unique
                new FieldMapGateway('unifonic', ['AppSid', 'Body', 'Recipient'], ['SenderID'], ['Recipient'], ['Body']),
                // Smsc: login+psw+phones triple
                new FieldMapGateway('smsc', ['login', 'psw', 'phones'], ['sender'], ['phones'], ['mes']),
                // Isendpro: keyid+num+sms triple
                new FieldMapGateway('isendpro', ['keyid', 'num', 'sms'], ['emetteur'], ['num'], ['sms']),
                // Yunpian: apikey+mobile pair
                new FieldMapGateway('yunpian', ['apikey', 'mobile'], ['from'], ['mobile'], ['text']),
                // SimpleTextin: contactPhone is unique
                new FieldMapGateway('simpletextin', ['contactPhone', 'text'], ['accountPhone'], ['contactPhone'], ['text']),
                // Sendberry: key+name+content triple
                new FieldMapGateway('sendberry', ['key', 'name', 'content'], ['from'], ['to'], ['content']),
                // TurboSms: sms object field is unique
                new FieldMapGateway('turbosms', ['sms'], ['sender'], ['recipients'], ['text']),
                // ClickSend: messages array (but different from infobip)
                new FieldMapGateway('clicksend', ['messages'], ['from'], ['to'], ['body']),
                // Smsmode: body+recipient nested
                new FieldMapGateway('smsmode', ['body', 'recipient'], ['from'], ['recipient'], ['body']),

                // ===== Russian providers =====

                // SMS.ru: api_id is unique
                new FieldMapGateway('smsru', ['api_id', 'msg'], ['from'], ['to'], ['msg']),
                // SMS Aero: sign+number combo
                new FieldMapGateway('smsaero', ['sign', 'number'], ['sign'], ['number'], ['text']),
                // Devino Telecom: SourceAddress+DestinationAddress+Data
                new FieldMapGateway('devino', ['SourceAddress', 'DestinationAddress', 'Data'], ['SourceAddress'], ['DestinationAddress'], ['Data']),
                // IQSms: clientId is unique
                new FieldMapGateway('iqsms', ['clientId', 'phone'], ['sender'], ['phone'], ['text']),
                // MTS Communicator: naming+msisdn combo
                new FieldMapGateway('mts', ['naming', 'msisdn'], ['naming'], ['msisdn'], ['text']),
                // Beeline SMS: target is unique field name
                new FieldMapGateway('beeline', ['target', 'message'], ['sender'], ['target'], ['message']),
                // Megafon: subject+message combo
                new FieldMapGateway('megafon', ['subject', 'message'], ['from'], ['to'], ['message']),

                // ===== Less unique — need multiple fields to distinguish =====

                // SevenIo: json flag is somewhat unique
                new FieldMapGateway('sevenio', ['json', 'text', 'to'], ['from'], ['to'], ['text']),
                // GatewayApi: recipients array + message
                new FieldMapGateway('gatewayapi', ['recipients', 'message', 'sender'], ['sender'], ['recipients'], ['message']),
                // Redlink: phoneNumbers is unique
                new FieldMapGateway('redlink', ['phoneNumbers', 'message'], ['sender'], ['phoneNumbers'], ['message']),
                // OvhCloud: receivers is unique
                new FieldMapGateway('ovhcloud', ['receivers', 'message'], ['sender'], ['receivers'], ['message']),
                // Primotexto: number+message combo
                new FieldMapGateway('primotexto', ['number', 'message', 'from'], ['from'], ['number'], ['message']),
                // Mobyt: message_type is unique
                new FieldMapGateway('mobyt', ['message_type', 'message'], ['sender'], ['recipient'], ['message']),
                // Smsapi: format+encoding combo
                new FieldMapGateway('smsapi', ['format', 'encoding', 'message'], ['from'], ['to'], ['message']),
                // Octopush: sms_text is unique
                new FieldMapGateway('octopush', ['sms_text', 'sms_recipients'], ['sms_sender'], ['sms_recipients'], ['sms_text']),
                // SmsBiuras: uid+apikey combo
                new FieldMapGateway('smsbiuras', ['uid', 'apikey', 'message'], ['from'], ['to'], ['message']),
                // FortySixElks: from+to+message (very generic, keep near bottom)
                new FieldMapGateway('46elks', ['from', 'to', 'message'], ['from'], ['to'], ['message']),
                // Clickatell: from+to+text (generic, keep near bottom)
                new FieldMapGateway('clickatell', ['from', 'to', 'text'], ['from'], ['to'], ['text']),
                // AllMySms: same as clickatell
                new FieldMapGateway('allmysms', ['from', 'to', 'text'], ['from'], ['to'], ['text']),
                // RingCentral: from+to+text (generic)
                new FieldMapGateway('ringcentral', ['from', 'to', 'text'], ['from'], ['to'], ['text']),

                // Generic fallback: tries common field names, always matches
                new FieldMapGateway(
                    gatewayName: 'generic',
                    detectFields: [],
                    fromFields: ['from', 'From', 'sender', 'Sender', 'originator', 'msisdn', 'SenderID', 'source_number', 'sign', 'naming', 'SourceAddress', 'sms_sender', 'emetteur', 'sender_id'],
                    toFields: ['to', 'To', 'recipient', 'Recipient', 'dst', 'phone', 'phones', 'mobile', 'num', 'destination_number', 'contactPhone', 'number', 'target', 'DestinationAddress', 'sms_recipients', 'receivers', 'phoneNumbers'],
                    messageFields: ['message', 'body', 'Body', 'text', 'Text', 'content', 'Content', 'sms', 'mes', 'msg', 'Data', 'sms_text'],
                ),
            ]),
        ];
    }

    public function boot(EventTypeRegistryInterface $registry): void
    {
        $registry->register('sms', new EventTypeMapper());
    }
}
