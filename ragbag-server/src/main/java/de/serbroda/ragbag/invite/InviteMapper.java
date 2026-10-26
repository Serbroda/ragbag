package de.serbroda.ragbag.invite;

import de.serbroda.ragbag.generated.model.InviteDto;
import de.serbroda.ragbag.generated.model.InvitePreviewDto;
import de.serbroda.ragbag.generated.model.InviteRoleEnum;
import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import org.springframework.stereotype.Component;

@Component
public class InviteMapper {

    public InviteDto toDto(Invite invite) {
        return new InviteDto()
                .token(invite.getToken())
                .targetType(
                        InviteDto.TargetTypeEnum.valueOf(invite.getTargetType().name()))
                .targetId(invite.getTargetId())
                .role(InviteRoleEnum.fromValue(invite.getRole()))
                .expiresAt(toOffsetDateTime(invite.getExpiresAt()))
                .maxUses(invite.getMaxUses() > -1 ? invite.getMaxUses() : null)
                .usedCount(invite.getUsedCount())
                .createdAt(toOffsetDateTime(invite.getCreatedAt()))
                .createdBy(invite.getCreatedBy().getUsername());
    }

    public InvitePreviewDto toDto(InviteService.InvitePreview preview) {
        Invite invite = preview.invite();
        return new InvitePreviewDto()
                .targetType(InvitePreviewDto.TargetTypeEnum.valueOf(
                        invite.getTargetType().name()))
                .targetId(invite.getTargetId())
                .targetName(preview.targetName())
                .role(InviteRoleEnum.fromValue(invite.getRole()))
                .invitedBy(preview.invitedBy())
                .valid(preview.valid())
                .alreadyMember(preview.alreadyMember());
    }

    private static OffsetDateTime toOffsetDateTime(Instant instant) {
        return instant != null ? instant.atOffset(ZoneOffset.UTC) : null;
    }
}
