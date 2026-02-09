package de.serbroda.ragbag.invite;

import de.serbroda.ragbag.generated.model.InviteDto;
import org.springframework.stereotype.Component;

@Component
public class InviteMapper {

    public InviteDto toDto(Invite invite) {
        return new InviteDto()
                .token(invite.getToken())
                .targetType(
                        InviteDto.TargetTypeEnum.valueOf(invite.getTargetType().name()))
                .targetId(invite.getTargetId())
                .expiresAt(invite.getExpiresAt())
                .maxUses(invite.getMaxUses())
                .usedCount(invite.getUsedCount());
    }
}
