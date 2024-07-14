<?php

declare(strict_types=1);

namespace Interfaces\Http\Sentry;

use Modules\Projects\Domain\Project;
use Modules\Projects\Domain\ValueObject\Key;
use Nyholm\Psr7\Stream;
use Tests\App\Http\ResponseAssertions;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

final class SentryVueReplayActionTest extends ControllerTestCase
{
    protected const JSON = <<<'BODY'
{"event_id":"53cbfd0ae9fd4ea783cec45310432a3c","sent_at":"2024-06-14T08:10:06.632Z","sdk":{"name":"sentry.javascript.vue","version":"8.9.2"}}
{"type":"replay_event"}
{"type":"replay_event","replay_start_timestamp":1718352515.3176,"timestamp":1718352606.631,"error_ids":["800d6a6ef3174aa5ba3f619204f9d1a9"],"trace_ids":[],"urls":["http://localhost:5173/"],"replay_id":"53cbfd0ae9fd4ea783cec45310432a3c","segment_id":0,"replay_type":"buffer","request":{"url":"http://localhost:5173/","headers":{"Referer":"http://localhost:5173/","User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"}},"event_id":"53cbfd0ae9fd4ea783cec45310432a3c","environment":"production","sdk":{"integrations":["InboundFilters","FunctionToString","BrowserApiErrors","Breadcrumbs","GlobalHandlers","LinkedErrors","Dedupe","HttpContext","Vue","BrowserTracing","Replay","ReplayCanvas"],"name":"sentry.javascript.vue","version":"8.9.2"},"transaction":"home","platform":"javascript"}
{"type":"replay_recording","length":5315}
{"segment_id":0}
xÜ\ërÛ6~:Í¤]Q)ëÉì¶étmffû£ît 	S¤d{3~÷ý xE9VÛT¶eÎýà@úõc'½ßÓÎüªÛYtæ;Û®;óÎ6M÷ó~?dKnYÎGÞdØït;·Á*Ývæ7t;[l¶)îÞô¡ÛIMR²Û£eâM#<¼«®hdï£
0`û4`Q)÷ä>ddÅ%[vWoéÅ«W$:¤3Oãív$èÿCú¤lÐóºÇ,6[ÑvHè+¶ÛÇÙ7b"go¾	Ãé*6Yã$¼ígzbM¯£ý!ÕH«Ö¦·,¾ù¦$ Éû8Ä5	üá+²O1ý­ªÁÉgoé`Fþ@ÉÆÕçÉ¢å9ãý½[¡A=Û \ýVÓ*6Ed¨Â.äR8,Â`ùbèà.¹ORºSwþûÉ Ã
ÃIÆÁâòI>vBqÓ¨µ©¿da´¸Ï+/·$N($Ùyÿó÷î´ÂoA#AaÝT`ÆñïÒ2:ý£AYh±L2¶LR¸È1 ·{§x¾dQJ#>ýå%uÅM×	¢
Hè&0búÒë
lXL,X¤AÒ¶b$äFék¼üI=ÈÙà*jgKÒûêl²£Ý_&ÜpEvAJ]ër~ÌXÚé%}²ß'ýãºû} Ë´ÄË>I ñ¤¿#AÔãPZ1çëSwéÞn1ïÜùb-^/
ÍnÂÖ)6å?¥g;ÐÃùüG?[dyfoÊÍ
/^¥g?å?úY­
ãíË!
tû*8ðnÈÒõæN¼YçãA×Q?ûÊÒß¯íïùÕþ+ßhðÓ«®£þ½ñÈÒ]A/v¿æÝ¹@rÔ$~^ ·¶c=Îãq©³BØ*ÄU×Mô£®¿êº?ä²d!ÝÄ´Ù!ZYÀ»IyûæúTðfsb%ýJò­tv·ìØ8Ä3pçDBoSdFWÞÜÜ/Á"tº²;Þ¬ý×cç9ëÑMK[ºý\8_9¯#Çi)aB-ebKäöÙNn#LiÛH Ò³^ëèáë®3/èÅ_uJc°tÁîÜ$øFb¦ÎÄ ;BRk¸M÷V½¹±xGB]°Õ= ìfÏ¼Áà¸}á¤>9À¤*¹2çÑ)I Ä1Ì/ÃGCIBAÒ5d£_8ä4Ç¨7ø¯É.ïçÎk¸ë¸'!¬P$<]ç[þ$Ëwâþ{è:×wtÃ¨óþõu§ë¼e²®óæî~C£®ó~qÒC×A¾!ð}çÐç#®;ßÅ,X÷?ÐðHÓ`Ièò¶Ï`q #Ð0 xÞó]H1¦¤"Äsçt,0Hï¡·tq<1xCÙ¾À
9B¦¬ «/@-¹ËÌàOl
A;ä²Î¬2µócº³	tÞ&¦4ÔLÙxòN2¡E,¢Zà\ÃK{ÓÞ|î©Mù0äaÁ~XpÒnÄ?éÈ\Üq-.itéÆ´"JSP ¹æ*nü«ý+{$û@cÖ!&¸F
G¼K È"h,À9d®î¿ð¿Cí±BÁú£uØEé­cþgÐÉ9±#Ç³ªäHärgB§²,oð'¤YK¬~ Ó©T8d¿°8\õÐëø{Æçx)¦ÐMï^%K¶§«ÔP2xÏøÂà¼mëýaê* ¿Çe

Fã÷Æ CjÌ*y2¦Çõ²À³ZaKóëIO!¦Ä©AHÚ|^TØìpÃ}RS
ø¤³t6ªÖP«m­wIÕúf¿o¡H2!ÃÉâQ$ÖÎK
çR)P¸3¼+Ç²^È6¬n¸¶w±·Põ¤Ê¨#r¬®?ø²èú}íúKzMà
UÎa;¤ÏH¸¢Þ%F"Ý:Òºií±ù\`
ÞWxø=¢bÄUOÀmä`	YHF<£'©	×epÍ	Y)è¢é¨¢»â$uÙÚå"R,y+&£6´«Q8!IrãL·°p_>Ï72ã¯¾#c|MíL5ãZ .2r<Þms;aÞäb/àÄ53wø»À¢ÎU#Õ5ÉÑÔóÌW+YóÛSÍ³N
Îk¤±ùgÄÅ_h;úlláÏ«Ùb¸áÏz\RJ
N®¦{(º&.>ôVbÿ2©ÈAøá>ª O) nX+r+o7>Pyâì®ÁqåA°}5WÓIRûí`T½+Ñ±f2¹ãóßmD$t=wubbgÙAÿPËyX
¾t\ÇÇ2¤	úc1¥Ö²A(÷zÒ³¶Z½IW[|ÊçL	X³»ÌÙÕ2L­WEÔrÝÁïyñ àl @@^³ím
Ñ*CVKëâ\ó'"\ÏµgË¹&!1!(TØ¼øD{ñ±º2ÜyÅrLäî3/PÕº}¾{ô7@OáÛV\Ø¸©Ä7+t@áû¼Òzn5«µð"|eâRHn².ØmªÈd5Uñ¸>Ê Y¥áÛÂrójhTLhÚQÕ]­2èSêeVËr×nN¼½RVWwáe£ôâOµø¶RÏvØ§Ö8ðÉ%ú¥nã¿I$6ó©Âj¹eT®®yö!AÖräek$ªQòûû³¢~Ô·C5hKmUKaì+¶½/zÕáÓxà²xóä:D7mN¡öË¾®÷ù¶"'MQ&ÿU+©^
#­ûCîc¥òdnÒ*É¢ç8uÌB¾¹ÈÎ×mÝÍ'â.1_Ocðlè£ûJï"^mgÐI Å öL1+dõ¾ÔUn¹òiömÆÇÂOª
1Û	 £R¥bàÉTË(ÝðÙX+öÇjN`$~õ(7±¤É»ÿüÛLõ®f
îFâ5hÉÛLûlVËUnË@n
Âî¡¨õ¯^j@kV}´ÑÚÛpÙØ®<þÈÆúªøÃÂW5÷Ð¬Òad(µ-}É©ü¢æAáâËØð66µå×-?7¯1¶n´Õx
3ÃÁúÞÿ]w·âÈNÛ'%§"Ú8±á­hëfl£S7ùÓ¢LZhîDëë¸ÉTï&£ÍË°¦ÁÖ	öI´}I°9à(EM	?uôä!Å¬&ÚóNShLQÀlÞá$ñK Ü;ÉçNÓÃ*`=lC_Âó*LuÞ=1"DÃâhbhèy¤mt{Xz>°m°»½éYHøÈ(j6YL!B(ý.(½!lCùeÌ¶Ís;···½å½8ÜØEb­ÂñJ3¤))eËOÅp3§JX.A^.'u¥°×ûÓ¦)§Öº	YÙ§³lMÜxY©ÊYáGf¶u·O:ñkòg¦d³¦
«&µQ}´5÷Ç6ãøÜr¦->±ÒòP=)'E¶Sçå¤hÖ"ÏtÚ<{²ôô/¡×f©¥{·POo õiÓ¿)'-®,êMºKtÁcðèTa^¾ØòLã¨oÐ%T=:KÈöÍ.HÂÔ8gÅk Íz®ê$mhMÝC
sº4}½U)4»fÌ÷pãÓ3=Ag¹|Ê¼Èh-×Q|lîÓ¼ÂÆd-78)ä&8bââîRéÂ5OÍæÔ¤L¯ßÆ9¢¸©ýiöwÙWEAôÉ6V[{qnÃø\ÛSýëF|³Êx:â7Öõt§\IQôjÑÿäâÈÛ
û1ëqgîlIzé0¡±Ô»iJmw?J>²H÷£÷qZqyÃ¢®C%1Ü¥ø\-ÿq`6tÅýgïÃü¥Ðù´úd;¶bÉ3ªt%Dôöâ)CY'ò2Bãy¦P!¨ü=6_º
Rêy¹}0e@MnÓ`ÒE_Þ%ÒS´qé¨i7§ÝÖªègLQ
üã)ê6§9ÖÛòå³¦(*¦f¶ØXVÔr=mSØôNò¤ÝcKÉ¾ª{|%c«ßªð5.^æ>X|/U-mazy
·mµL5wÛ¬RAñLK1¿4OØêEe£¿ÔÝóMÆX8ªé¡Ê×Üõ:ºæB²Î¹Fê¸ÓÈ/Zx³^ïÀ÷[à®øn
ÀÅ|í{O&ÿgïwÛ8þ*Ó>oUq~éSìÖH*mª*Óöö;¸8#æÿ:Ñ/i)8ÃÝ/¡-£ãýí°ßIP8Z_Þ?îÐyóåó_oD´J©ô¶ú¯+=Ý¼«­ÏkïE:Átóªí y¥ß}ÞUsåµ'¹b}jÞGùôyåó<¯8Ç¼õ1¯ÈrjK^³Yãòn[}»m3'¹f­m[mZºyô^ÆËÕ>[=¶mòªg¤Ëk5ocz	ç³j74îS]ÌºêÊ[+ü&\ÞçÎµòó¼ÏÞÝþðëYÐÃËk²e¼|?üôWjúãm/ª>¼Þ½¼í²¸ð(ºåÌÿïØ+®©¾ì{ï×r<ÚÈ
¯Ø83k´{³^IhE\ÒùJD%ÉDüÊp£c#µiä9ÿÈHy¡Ú	ÇíRþÝy÷©ÿú§'ñ¡ÿíÃÚÀ.ÔFwÉ_'gËä>®WZ&Ðé}4S~ç·Eþ§×¸È/çãÒ[ö^]¤Ï7?>0Á:~IÕ6´.Ù *Åj¹+Äç~pK¦nüÌ[$Ô*n_èsÆ¨ºáN¼J±Ã´ÑX N4Òé¡påcU>VåcU>VåcU>VåcU>Våc
âcSËßtêWÙY5?^r,;öºY¹Z«U¹Z«5/®XØüE7HNü+s«2·*s«2·*skþÌ-°{ßîý!IsÉGã4*pX¨p?Âo?EwèeÓb@^q£Â¸¿õëºÚûAtv·u]äP´¿g÷`@ïÒÓ°3»,ôÇ¯&Éúl.ÓÙóÅc@w<+VÙðIf4b¤^<&Å
KL(n^Eóâ¼sÓÿKàÎ@ón!$4k¼½^(4 AÆðîË ýVûâ¦(ÞlD(ê%ÈvÅrÑÄ|24Nsv 6 ;^ÜL ;6¤
ÔBàd7GR~ÑVq³Â0Ù ÞPÛR¿hJ¬nD(8xÞ(¤g} Do Òq
è,ü}ÔGÕNÚ)rËWÐ6Ì
,ÂIô&of ]Ò	Å°+Àëúd>\Ýí@ÅÓ`îbÜ
ÒS&(ïÎ», h¾,0iÀ1éz%Ðäe	ÇÑAfÄ^RÙ)ÄhX=P'90HÆEýsÐbP<þ'Áýhð³²ÁG÷
NJ:é·Â=MëAA&Ó,GE%ÈÔÈ¤?¹CÎV+ÀµrÂÄËp±¬bnB® PnIæ2ëbÛë
øÁX}Å1ôtG9qù:BÐ²'éÌüBPÃxÇi` 	äÔ<Iz>Ã²W~q
Á{Ñî&HÈq³äkR¶Ð Ãþ%Èì+qYyzp4¯rjPLPÝK
}-@$ä¹ÐÊøÄÛ¦Ù}¸@fñ]Õ<  EøeÊ&ñÞñíºbþåÄ¬G :±V|:Ê(±¨´=K$$PÔ[á
ÝÖ¬"¿Á²K^Æ¨Ê¢-Û®bØÞSg2ÿBÛ²HløíÈC_lÒÀÃv(±9Ïãïî¼cS-bÅ%×@\ýñM:°¾êÝcÅ}×NÒC³5cäv%±h¾ÔÂÃ,l{07AIuÂJÿ¿À¶ó9·rUõ¯³3ÊmÁð]z
-iaÚ{Ýëª©SÕ¤íüûÿØI!!h§ª±ìÄ¡!ö×­|NÓ(Ã²
®'|sJ\aB6	òö?gÜ<ï¯B	ÊaF¬,Éê¹eî<3;3'³qC![qóPö h@%6mKk@
GPçË>ÒJÃbnÒYUã[D·n)Ìútð²Ûõº[AZ¹nM+Ýã´'¹n¦°kìr¯dÌ5_Vå¦Öä4ÝO¶Tb¤H¶|»ü`±Àþ»\ç¿½2Ò&*P7ô_ÇSwí¹L:Å{|¼+{¦ßàIrü>+1q'5vg2Q^¢8Hø
ìóÉ¬5Þýö¼·|®ÕËáx¢Æeñëbá/$ø1Þ83®f!ZnïlõÎpÔ5ñë?¢½Ewd½QuøôIÈãµAâÁ·x_VWp gï0=uA«¸³ØIã¶ò}Öb+8{^¯Ù1®
BODY;

    private Project $project;

    protected function setUp(): void
    {
        parent::setUp();

        $this->project = $this->createProject('default');
    }

    public function testSend(): void
    {
        $this->makeRequest(project: $this->project->getKey())->assertOk();

        $this->markTestIncomplete('This test has not been implemented yet.');
    }

    private function makeRequest(string $secret = 'secret', string|Key $project = 'default'): ResponseAssertions
    {
        return $this->http
            ->postJson(
                uri: '/api/' . $project . '/envelope/?sentry_key=' . $secret . '&sentry_version=7&sentry_client=sentry.javascript.vue%2F8.9.2',
                data: Stream::create(self::JSON),
                headers: [
                    'User-Agent' => 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36',
                ],
            );
    }
}
